# dn42regsrv API Description

## Route Origin Authorisation (ROA) API

Route Origin Authorisation (ROA) data can be obtained from the server in
JSON and bird formats.

### JSON format output

```
GET /api/roa/json
```

Provides IPv4 and IPv6 ROAs in JSON format, suitable for use with
[gortr](https://github.com/cloudflare/gortr). 

Example Output:
```
wget -O - -q http://localhost:8042/api/roa/json | jq
```

```
{
  "metadata": {
    "counts": 1564,
    "generated": 1550402199,
    "valid": 1550445399
  },
  "roas": [
    {
      "prefix": "172.23.128.0/26",
      "maxLength": 29,
      "asn": "AS4242422747"
    },
    {
      "prefix": "172.22.129.192/26",
      "maxLength": 29,
      "asn": "AS4242423976"
    },
    {
      "prefix": "10.110.0.0/16",
      "maxLength": 24,
      "asn": "AS65110"
    },

... and so on
```

### Bird format output

```
GET /api/roa/bird/{bird version}/{IP family}
```

Provides ROA data suitable for including in to bird.

{bird version} must be either 1 or 2

{IP family} can be 4, 6 or 46 to provide both IPv4 and IPv6 results


Example Output:
```
wget -O - -q http://localhost:8042/api/roa/bird/1/4
```

```
#
# dn42regsrv ROA Generator
# Last Updated: 2019-02-17 11:16:39.668799525 +0000 GMT m=+0.279049704
# Commit: 3cbc349bf770493c016888ff785227ded2a7d866
#
roa 172.23.128.0/26 max 29 as 4242422747;
roa 172.22.129.192/26 max 29 as 4242423976;
roa 10.110.0.0/16 max 24 as 65110;
roa 172.20.164.0/26 max 29 as 4242423023;
roa 172.20.135.200/29 max 29 as 4242420448;
roa 10.65.0.0/20 max 24 as 4242420420;
roa 172.20.149.136/29 max 29 as 4242420234;
roa 10.160.0.0/13 max 24 as 65079;
roa 10.169.0.0/16 max 24 as 65534;

... and so on
```

```
wget -O - -q http://localhost:8042/api/roa/bird/2/6
```

```
#
# dn42regsrv ROA Generator
# Last Updated: 2019-02-17 11:16:39.668799525 +0000 GMT m=+0.279049704
# Commit: 3cbc349bf770493c016888ff785227ded2a7d866
#
route fdc3:10cd:ae9d::/48 max 64 as 4242420789;
route fd41:9805:7b69:4000::/51 max 64 as 4242420846;
route fd41:9805:7b69:4000::/51 max 64 as 4242420845;
route fd41:9805:7b69:4000::/51 max 64 as 4242420847;
route fddf:ebfd:a801:2331::/64 max 64 as 65530;
route fd42:1a2b:de57::/48 max 64 as 4242422454;
route fd42:7879:7879::/48 max 64 as 4242421787;

... and so on
```

## Registry API

The general form of the registry query API is:

```
GET /api/registry/{type}/{object}/{key}/{attribute}?raw
```

* Prefixing with a '*' performs a case insensitive, substring match
* A '*' on its own means match everything
* Otherwise an exact, case sensitive match is performed

By default, results are returned as JSON objects, and the registry data is decorated
with markdown style links depending on relations defined in the DN42 schema. For object
results, a 'Backlinks' section is also added providing an array of registry objects that
reference this one.

If the 'raw' parameter is provided, attributes are returned un-decorated exactly
as contained in the registry.

Some examples will help clarify:

* Return a JSON object, with keys for each registry type and values containing a count
of the number of registry objects for each type

```
wget -O - -q http://localhost:8042/api/registry/ | jq
{
  "as-block": 8,
  "as-set": 34,
  "aut-num": 1486,
  "domain": 451,
  "inet6num": 746,
  "inetnum": 1276,
  "key-cert": 7,
  "mntner": 1379,
  "organisation": 275,
  "person": 1388,
  "registry": 4,
  "role": 14,
  "route": 892,
  "route-set": 2,
  "route6": 596,
  "schema": 18,
  "tinc-key": 25,
  "tinc-keyset": 3
}

```

* Return a list of all objects in the role type

```
wget -O - -q http://localhost:8042/api/registry/role | jq
{
  "role": [
    "ORG-NETRAVNEN-DN42",
    "PACKETPUSHERS-DN42",
    "CCCKC-DN42",
    "FLHB-ABUSE-DN42",
    "NIXNODES-DN42",
    "ORG-SHACK-ABUSE-DN42",
    "ORG-SHACK-TECH-DN42",
    "ORG-YANE-DN42",
    "SOURIS-DN42",
    "CCCHB-ABUSE-DN42",
    "MAGLAB-DN42",
    "NL-ZUID-DN42",
    "ORG-SHACK-ADMIN-DN42",
    "ALENAN-DN42"
  ]
}
```

* Returns a list of all objects in types that match 'route'

```
wget -O - -q http://localhost:8042/api/registry/*route | jq
{
  "route": [
    "172.20.28.0_27",
    "172.23.220.0_24",
    "172.23.82.0_25",
    "10.149.0.0_16",

...

    "172.20.128.0_27",
    "172.22.127.32_27"
  ],
  "route-set": [
    "RS-DN42",
    "RS-DN42-NATIVE"
  ],
  "route6": [
    "fd42:df42::_48",
    "fd5c:0f0f:39fc::_48",

...

    "fd16:c638:3d7c::_48",
    "fd23::_48"
  ]
}
```

* Returns the mntner/BURBLE-MNT object (in decorated format)

```
wget -O - -q http://localhost:8042/api/registry/mntner/BURBLE-MNT | jq
{
  "mntner/BURBLE-MNT": {
    "Attributes": [
      [
        "mntner",
        "BURBLE-MNT"
      ],
      [
        "descr",
        "burble.dn42 https://dn42.burble.com/"
      ],
      [
        "admin-c",
        "[BURBLE-DN42](person/BURBLE-DN42)"
      ],
      [
        "tech-c",
        "[BURBLE-DN42](person/BURBLE-DN42)"
      ],
      [
        "auth",
        "pgp-fingerprint 1C08F282095CCDA432AECC657B9FE8780CFB6593"
      ],
      [
        "mnt-by",
        "[BURBLE-MNT](mntner/BURBLE-MNT)"
      ],
      [
        "source",
        "[DN42](registry/DN42)"
      ]
    ],
    "Backlinks": [
      "aut-num/AS4242422602",
      "aut-num/AS4242422601",
      "mntner/BURBLE-MNT",
      "route/172.20.129.160_27",
      "as-set/AS4242422601:AS-DOWNSTREAM",
      "as-set/AS4242422601:AS-TRANSIT",
      "person/BURBLE-DN42",
      "inet6num/fd42:4242:2601::_48",
      "domain/burble.dn42",
      "domain/collector.dn42",
      "route6/fd42:4242:2601::_48",
      "inetnum/172.20.129.160_27"
    ]
  }
}
```

* Returns error 404, exact searches are case sensitive

```
wget -O - -q http://localhost:8042/api/registry/mntner/burble-mnt | jq
```

* Returns domain names matching 'burble' in raw format

```
wget -O - -q http://localhost:8042/api/registry/domain/*burble?raw | jq
{
  "domain/burble.dn42": [
    [
      "domain",
      "burble.dn42"
    ],
    [
      "descr",
      "burble.dn42 https://dn42.burble.com/"
    ],
    [
      "admin-c",
      "BURBLE-DN42"
    ],
    [
      "tech-c",
      "BURBLE-DN42"
    ],
    [
      "mnt-by",
      "BURBLE-MNT"
    ],
    [
      "nserver",
      "ns1.burble.dn42 172.20.129.161"
    ],
    [
      "nserver",
      "ns1.burble.dn42 fd42:4242:2601:ac53::1"
    ],
    [
      "ds-rdata",
      "61857 13 2 bd35e3efe3325d2029fb652e01604a48b677cc2f44226eeabee54b456c67680c"
    ],
    [
      "source",
      "DN42"
    ]
  ]
}
```

* Returns all objects matching 172.20.0

```
wget -O - -q http://localhost:8042/api/registry/*/*172.20.0 | jq
{
  "inetnum/172.20.0.0_14": {
    "Attributes": [
      [
        "inetnum",
        "172.20.0.0 - 172.23.255.255"
      ],
      [
        "cidr",
        "172.20.0.0/14"
      ],

... and so on
```

* Returns the nic-hdl attribute for all person objects

```
wget -O - -q http://localhost:8042/api/registry/person/*/nic-hdl | jq
{
  "person/0RIGO-DN42": {
    "nic-hdl": [
      "0RIGO-DN42"
    ]
  },
  "person/0XDRAGON-DN42": {
    "nic-hdl": [
      "0XDRAGON-DN42"
    ]
  },
  "person/1714-DN42": {
    "nic-hdl": [
      "1714-DN42"
    ]
  },

... and so on
```

* return raw contact (-c) attributes in aut-num objects that contain 'burble'

```
wget -O - -q http://localhost:8042/api/registry/aut-num/*/*-c/*burble?raw | jq
{
  "aut-num/AS4242422601": {
    "admin-c": [
      "BURBLE-DN42"
    ],
    "tech-c": [
      "BURBLE-DN42"
    ]
  },
  "aut-num/AS4242422602": {
    "admin-c": [
      "BURBLE-DN42"
    ],
    "tech-c": [
      "BURBLE-DN42"
    ]
  }
}
```

## DNS Root Zone API

The DNS API provides a list of resource records that can be used to create a root zone for DN42
related domains. By polling the API, DNS servers are able to keep their root zone delegations and
DNSSEC records up to date.


```
GET /api/dns/root-zone?format={[json|bind]}
```

Format may either 'json' or 'bind' to provide resource records in either format. The default
output format is JSON.

Example Output (JSON format):
```
wget -O - -q http://localhost:8042/api/dns/root-zone?format=json | jq
```

```
{
  "Records": [
    {
      "Name": "dn42",
      "Type": "NS",
      "Content": "b.delegation-servers.dn42.",
      "Comment": "DN42 Authoritative Zone"
    },
    {
      "Name": "dn42",
      "Type": "NS",
      "Content": "j.delegation-servers.dn42.",
      "Comment": "DN42 Authoritative Zone"
    },

... and so on
```

Example Output (BIND format):
```
wget -O - -q http://localhost:8042/api/dns/root-zone?format=bind
```

```
;; DN42 Root Zone Records
;; Commit Reference: 2cc95d9101268ce82239dee1f947e4a8273524a9
;; Generated: 2019-03-08 19:40:51.264803795 +0000 GMT m=+0.197704585
dn42    IN      NS      b.delegation-servers.dn42.      ; DN42 Authoritative Zone
dn42    IN      NS      j.delegation-servers.dn42.      ; DN42 Authoritative Zone
dn42    IN      NS      y.delegation-servers.dn42.      ; DN42 Authoritative Zone
dn42    IN      DS      64441 10 2 6dadda00f5986bd26fe4f162669742cf7eba07d212b525acac9840ee06cb2799   ; DN42 Authoritative Zone
dn42    IN      DS      56676 10 2 4b559c949eb796f5502f05bd5bb2143672e7ef935286db552955f291bb81093e   ; DN42 Authoritative Zone
d.f.ip6.arpa    IN      NS      b.delegation-servers.dn42.      ; DN42 Authoritative Zone
d.f.ip6.arpa    IN      NS      j.delegation-servers.dn42.      ; DN42 Authoritative Zone
d.f.ip6.arpa    IN      NS      y.delegation-servers.dn42.      ; DN42 Authoritative Zone
d.f.ip6.arpa    IN      DS      64441 10 2 9057500a3b6e09bf45a60ed8891f2e649c6812d5d149c45a3c560fa0a619
5c49    ; DN42 Authoritative Zone
d.f.ip6.arpa    IN      DS      56676 10 2 d93cfd941025aaa445283d33e27157bb9a2df0a9c1389fdf5e36a377fc31
4736    ; DN42 Authoritative Zone

... and so on
```