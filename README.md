# dn42regsrv

A REST API for the DN42 registry, written in Go, to provide a bridge between
interactive applications and registry data.

A public instance of the API and explorer web app can be accessed via:

* [https://explorer.burble.com/](https://explorer.burble.com/) (Internet link)
* [http://explorer.collector.dn42/](http://explorer.collector.dn42/) (DN42 Link)

## Features

* REST API for querying DN42 registry objects
* Able to decorate objects with relationship information based on SCHEMA type definitions
* Includes a simple webserver for delivering static files which can be used to deliver
  basic web applications utilising the API (such as the included DN42 Registry Explorer)
* Automatic pull from the DN42 git repository to keep the registry up to date
* Includes a responsive web app for exploring the registry
* API endpoints for ROA data in JSON, and bird formats
* API endpoint to support the creation of DNS root zone records

## Building

#### Using locally installed go

Requires [git](https://git-scm.com/) and [go](https://golang.org)  
```
go get -insecure git.dn42.us/burble/dn42regsrv
```

#### Without installing go

Using container runtime to build with the golang container:  
```
docker run -v ${PWD}:/go/bin golang go get -insecure git.dn42.us/burble/dn42regsrv
```

Or use the *contrib/build.sh* script after cloning the repo.

## Running

#### As a service

Use --help to view configurable options
```
${GOPATH}/bin/dn42regsrv --help
```

The server requires access to a clone of the DN42 registry and for
the git executable to be accessible.  
If you want to use the auto pull feature then the registry must
also be writable by the server.

```
cd ${GOPTH}/src/git.dn42.us/burble/dn42regsrv
git clone http://git.dn42.us/dn42/registry.git
${GOPATH}/dn42regsrv
```

A sample service file is included for running the server under systemd

#### Within a container

A container build script (*contrib/buildah.sh*) is included in the
contrib directory. The script uses [buildah](https://buildah.io/).

See the *contrib/entrypoint.sh* script for environment variables that can
be set when running the container.

## Using

By default the server will be listening on port 8042.  
See the [API.md](API.md) file for a detailed description of the API.


## Support

Please feel free to raise issues or create pull requests for the project git repository.

## #ToDo

### Server

- Add WHOIS interface

### DN42 Registry Explorer Web App

- Allow for attribute searches

