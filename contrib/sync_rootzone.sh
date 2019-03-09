#!/bin/bash
##########################################################################
#
# This script is intended to be run via cron job to sync a local,
# authoritative DNS server's root zone with the registry information
# provided by dn42regsrv.
#
# As is, the script is specific to updating PDNS within the burble.dn42
# network, however it is also intended to be easily adaptable to other
# DNS servers and networks
#
##########################################################################

##########################################################################
# This array is used to define additional, local networks that the
# DNS server may be authoritative for. The array is used to prevent
# the related resource records from being removed automatically,
# as any records not listed here or in the registry will get deleted
# make sure to include '.' here or the local NS and SOA records
# will get removed

IGNORE_RECORDS=(
    '.'
    '$ORIGIN'
    'ns1.burble.dn42'
    'burble.dn42'
    'collector.dn42'
    '1.0.6.2.2.4.2.4.2.4.d.f.ip6.arpa'
    '160/27.129.20.172.in-addr.arpa'
)

##########################################################################
# The functions here are used to actually update the DNS server
# Change these to use a different server than PDNS

PDNSUTIL='/usr/bin/pdnsutil'
PDNSCTRL='/usr/bin/pdns_control'

# replace_rr <name> <type> <content>
function replace_rr {
    local rr_name=$1; shift
    local rr_type=$1; shift
    local rr_content=$*

    >&2 echo "Replace: ${rr_name} ${rr_type} '${rr_content}'"
    
    if [ ${DEBUG} -eq 0 ]
    then
        ${PDNSUTIL} replace-rrset . ${rr_name} ${rr_type} "${rr_content}"
    fi
}


# delete_rr <name> <type>
function delete_rr {
    local rr_name=$1
    local rr_type=$2

    >&2 echo "Delete: ${rr_name} ${rr_type}"

    if [ ${DEBUG} -eq 0 ]
    then
        ${PDNSUTIL} delete-rrset . ${rr_name} ${rr_type}
    fi
}

# list the current contents of the root zone
function get_current_root_zone {
    local rra rr_name rr_type rr_content
    while read -r -a rra
    do
        rr_name=${rra[0]}
        rr_type=${rra[3]}
        rr_content=${rra[@]:4}

        current_rz["${rr_name} ${rr_type}"]=${rr_content}

    done < <(${PDNSUTIL} list-zone .)
}

# update the . SOA record after a change
function update_soa {
    >&2 echo "Incrementing SOA serial"
    if [ ${DEBUG} -eq 0 ]
    then
        ${PDNSUTIL} increase-serial .
    fi
}

# used to trigger a notify to any slaves of this server
function notify_slaves {
    >&2 echo "Notfying slaves"    
    if [ ${DEBUG} -eq 0 ]
    then    
        ${PDNSCTRL} notify .
    fi
}

##########################################################################
# No further local customisation should be needed from here
##########################################################################

# initialise script parameters and global vars

function usage {
    echo "Usage: $0 [-h] [-d] [-a URL] [-c FILE]"
    echo " -h: this help"
    echo " -d: enable debugging and don't action changes"
    echo " -a: URL to dn42regsrv API"
    echo " -c: file in which to store previous commit number"
}

# default options
DEBUG=0
APIURL="http://grc.burble.dn42:8043/api"
COMMITFILE="/tmp/.sync_rz_commit"

# parse any arguments passed to the script
while getopts ":hda:" opt
do
    case ${opt} in
        d)
            DEBUG=1
            ;;
        a)
            APIURL=${OPTARG}
            ;;
        *)
            usage
            exit 0
            ;;
    esac
done

# global vars
declare -A current_rz
declare -A new_rz
current_commit=''
new_commit=''

##########################################################################
# fetch and parse the root zone data from the API

function fetch_new_root_zone {
    local line fields rr_name rr_type rr_content
    while read -r line
    do
        if [[ ${line} == ';; Commit Reference:'* ]]
        then
            new_commit=${line#;; Commit Reference: }
        else
            # strip out comments and create array
            fields=( ${line%%;*} )

            # if the line is a valid record
            if [ ${#fields[@]} -ge 4 ]
            then
                rr_name=${fields[0]}
                rr_type=${fields[2]}
                rr_content=${fields[@]:3}
                new_rz["${rr_name} ${rr_type}"]=${rr_content}
            fi
        fi
    done < <(/usr/bin/wget -O - -q "${APIURL}/dns/root-zone?format=bind")

    if [ ${DEBUG} -eq 1 ]; then >&2 echo "New Commit: ${new_commit}"; fi
}

##########################################################################
# load and store the previous commit

function load_current_commit {
    read -r current_commit < "${COMMITFILE}" 
    if [ ${DEBUG} -eq 1 ]; then >&2 echo "Current Commit: ${current_commit}"; fi
}

function store_current_commit {
    if [ ${DEBUG} -eq 0 ]
    then
        echo "${1}" > "${COMMITFILE}"
    fi
}

##########################################################################
# remove records that have been deleted

function is_ignored {
    local rr_name=$1
    for i in "${IGNORE_RECORDS[@]}"
    do
        [ "${i}" == "${rr_name}" ] && echo "ignored"
    done
    echo ""
}

function remove_deleted_records {
    local key rr_name ignored count=0

    # check each record in the old root zone
    for key in "${!current_rz[@]}"
    do
        ignored=$(is_ignored ${key% })

        # if record is not ignored, and no new record exists
        if [ "${ignored}" == '' -a "${new_rz[${key}]}" == '' ]
        then
            # then get rid of it
            delete_rr ${key}
            count=$((count + 1))
        fi
    done

    echo ${count}
}

##########################################################################
# update records that have been added or changed

function update_new_records {
    local key content count=0

    # check each new record
    for key in "${!new_rz[@]}"
    do
	content="${new_rz[${key}]}"
	
        # if old record didn't exist, or content differs
        if [ "${current_rz[${key}]}" != "${content}" ]
        then
            # update the record
            replace_rr $key ${content}
            count=$((count + 1))            
        fi
        
    done

    echo ${count}
}

##########################################################################
# main flow of the script starts here

echo "DN42 Root Zone Sync"
date
echo

fetch_new_root_zone

# check that the commit was populated
if [ "${new_commit}" == '' ]
then
    echo "Unable to fetch new root zone, aborting"
    exit 1
fi

load_current_commit

# now check if anything actually needs to be done
if [ "${new_commit}" == "${current_commit}" ]
then
    echo "Commits are equal, nothing to do"
    exit 0
fi

get_current_root_zone

# apply changes
deleted_records=$(remove_deleted_records)
echo "Deleted ${deleted_records} records"
updated_records=$(update_new_records)
echo "Updated ${updated_records} records"

# bail out if there were no actual differences
if [ $((deleted_records + updated_records)) -eq 0 ]
then
    echo "No records were updated, exiting"
    exit 0
fi

# update the SOA and send out a notification to slaves
update_soa
notify_slaves

# finally store the new commit to show it's been updated
store_current_commit "${new_commit}"

echo "All done"

##########################################################################
# end of code
