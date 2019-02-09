# dn42regsrv API Description

## GET /&lt;file&gt;

If the StaticRoot configuration option points to a readable directory, files from
the directory will be served under /

The git repository contains a sample StaticRoot directory with a simple registry
explorer web app.

## GET /api/registry/

Returns a JSON object, with keys for each registry type and values containing a count
of the number of registry objects for each type.

Example:
```
http://localhost:8042/api/registry/

# sample output
{"as-block":8,"as-set":34,"aut-num":1482,"domain":451,"inet6num":744,"inetnum":1270,"key-cert":7,"mntner":1378,"organisation":275,"person":1387,"registry":4,"role":14,"route":886,"route-set":2,"route6":594,"schema":18,"tinc-key":25,"tinc-keyset":3}
```


## GET /api/registry/&lt;type&gt;?match

Returns a JSON object listing all objects for the matched types.

Keys for the returned object are registry types, the value for each type is an
array of object names

If the match parameter is provided, the &lt;type&gt; is substring matched against
all registry types, otherwise an exact type name is required.

A special type of '*' returns all types and objects in the registry.

Example:
```
http://localhost:8042/api/registry/aut-num      # list aut-num objects
http://localhost:8042/api/registry/*            # list all types and objects
http://localhost:8042/api/registry/route?match  # list route and route6 objects

# sample output
{"role":["ALENAN-DN42","FLHB-ABUSE-DN42","ORG-SHACK-ADMIN-DN42","PACKETPUSHERS-DN42","CCCHB-ABUSE-DN42","ORG-NETRAVNEN-DN42","ORG-SHACK-ABUSE-DN42","MAGLAB-DN42","NIXNODES-DN42","SOURIS-DN42","CCCKC-DN42","NL-ZUID-DN42","ORG-SHACK-TECH-DN42","ORG-YANE-DN42"]}

```

## GET /api/registry/&lt;type&gt;/&lt;object&gt;?match&amp;raw

Return a JSON object with the registry data for each matching object.

The keys for the object are the object paths in the form &lt;type&gt;/&lt;object name&gt;. The values depends on the raw parameter.

if the raw parameter is provided, the returned object consists of a single key 'Attributes'
which will be an array of key/value pairs exactly as held within the registry.

If the raw parameter is not provided, the returned Attributes are decorated with markdown
style links depending the relations defined in the DN42 schema. In addition a
'Backlinks' key is added which provides an array of registry objects that
reference this one.

If the match parameter is provided, the &lt;object&gt; is substring matched against all
objects in the &lt;type&gt;. Matching is case insensitive.

If the match parameter is not provided, an exact, case sensitive object name is required.

A special object of '*' returns all objects in the type

Example:
```
http://localhost:8042/api/registry/domain/burble.dn42?raw # return object in raw format
http://localhost:8042/api/registry/mntner/BURBLE-MNT      # return object in decorated format
http://localhost:8042/api/registry/aut-num/2601?match     # return all aut-num objects matching 2601
http://localhost:8042/api/registry/schema/*               # return all schema objects

# sample output (raw)
{"domain/burble.dn42":[["domain","burble.dn42"],["descr","burble.dn42 https://dn42.burble.com/"],["admin-c","BURBLE-DN42"],["tech-c","BURBLE-DN42"],["mnt-by","BURBLE-MNT"],["nserver","ns1.burble.dn42 172.20.129.161"],["nserver","ns1.burble.dn42 fd42:4242:2601:ac53::1"],["ds-rdata","61857 13 2 bd35e3efe3325d2029fb652e01604a48b677cc2f44226eeabee54b456c67680c"],["source","DN42"]]}

# sample output (decorated)
{"mntner/BURBLE-MNT":{"Attributes":[["mntner","BURBLE-MNT"],["descr","burble.dn42 https://dn42.burble.com/"],["admin-c","[BURBLE-DN42](person/BURBLE-DN42)"],["tech-c","[BURBLE-DN42](person/BURBLE-DN42)"],["auth","pgp-fingerprint 1C08F282095CCDA432AECC657B9FE8780CFB6593"],["mnt-by","[BURBLE-MNT](mntner/BURBLE-MNT)"],["source","[DN42](registry/DN42)"]],"Backlinks":["as-set/AS4242422601:AS-DOWNSTREAM","as-set/AS4242422601:AS-TRANSIT","inetnum/172.20.129.160_27","person/BURBLE-DN42","route/172.20.129.160_27","inet6num/fd42:4242:2601::_48","mntner/BURBLE-MNT","aut-num/AS4242422601","aut-num/AS4242422602","route6/fd42:4242:2601::_48","domain/collector.dn42","domain/burble.dn42"]}}

```

