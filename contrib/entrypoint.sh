#!/bin/bash
##########################################################################

DN42REGSRV_WEBAPP=${DN42REGSRV_WEBAPP:-/data/webapp}
DN42REGSRV_REGDIR=${DN42REGSRV_REGDIR:-/data/registry}
DN42REGSRV_BRANCH=${DN42REGSRV_BRANCH:-master}
DN42REGSRV_BIND=${DN42REGSRV_BIND:-[::]:8042}
DN42REGSRV_INTERVAL=${DN42REGSRV_INTERVAL:-10m}
DN42REGSRV_LOGLVL=${DN42REGSRV_LOGLVL:-info}
DN42REGSRV_AUTOPULL=${DN42REGSRV_AUTOPULL:-true}

exec /app/dn42regsrv \
     -s "$DN42REGSRV_WEBAPP" \
     -d "$DN42REGSRV_REGDIR" \
     -p "$DN42REGSRV_BRANCH" \
     -b "$DN42REGSRV_BIND" \
     -i "$DN42REGSRV_INTERVAL" \
     -l "$DN42REGSRV_LOGLVL" \
     -a "$DN42REGSRV_AUTOPULL"   

##########################################################################
# end of file
