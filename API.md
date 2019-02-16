# dn42regsrv API Description

## Registry API

The general form of the registry query API is:

GET /api/registry/{type}/{object}/{key}/{attribute}?raw

* Prefixing with a '*' performs a case insensitive, substring match
* A '*' on its own means match everything
* Otherwise an exact, case sensitive match is performed

By default results are returned as JSON objects, and the registry data is decorated
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
