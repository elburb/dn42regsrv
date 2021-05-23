//////////////////////////////////////////////////////////////////////////
// DN42 IP Explorer
//////////////////////////////////////////////////////////////////////////

//////////////////////////////////////////////////////////////////////////
// root component

Vue.component('app-root', {
    template: '#app-root-template',

    methods: {
    }        

})

//////////////////////////////////////////////////////////////////////////
// free IPv4 view

Vue.component('app-free4', {
    template: '#app-free4-template',
    data() {
        return {

            state: 'invalid',
            error: '',
            inetnum: null,
            p4: [ ],
            free4: [ ],
            stats: {
                alloc: 0,
                addr: 0,
                nets: 0
            },
            
            filter: 27,
            filtered: [ ],
            ftotal: 0,

            asn: '4242423999',
            mntner: 'FOO',
            person: 'FOO',
            example: '0.0.0.0/0'
        }
    },
    methods: {

        // not used
        pretty4(msg, subnet, plen) {
            var octets = [ ]
            octets[0] = (subnet >> 24) & 0xFF
            octets[1] = (subnet >> 16) & 0xFF
            octets[2] = (subnet >> 8) & 0xFF
            octets[3] = subnet & 0xFF
            console.log(msg, ': ', octets.join("."),'/',plen)
        },

        // reload prefix data
        reload() {
            
            // reset current data
            this.inetnum = null
            this.p4.splice(0,this.p4.length)
            this.state = "loading"

            // IPv4 prefixes
            axios
                .get('/api/registry/inetnum/*')
                .then(response => {
                    this.inetnum = response.data
                    this.processIPv4()
                })
                .catch(error => {
                    this.error = error
                    this.state = 'error'
                    console.log(error)                    
                })
        },


        // parse an IPv4 string in to an object
        parse4(p) {
            p = p.replace('_','/')

            // split out prefix
            var s1 = p.split('/')
            // and octets
            var s2 = s1[0].split('.')
            if (s1.length !=2 || s2.length != 4) {
                console.log("Failed to IPv4 parse ", p)
                return null
            }

            // convert to a 32bit integer
            var num =
                ((parseInt(s2[0]) << 24) +
                 (parseInt(s2[1]) << 16) +
                 (parseInt(s2[2]) << 8) +
                 parseInt(s2[3]))>>>0

            var plen = parseInt(s1[1])

            var mask = (0xFFFFFFFF - ((2**(32 - plen))-1))>>>0

            // finally return the ipv4 object
            return {
                plen: plen,
                mask: mask,
                subnet: num
            }
        },
        
        // process IPv4 prefixes
        processIPv4() {

            var inets = [ ]
            Object.values(this.inetnum).forEach(rp => {

                // map the attributes in to a hash
                // no need to worry about duplicated attribs
                var attrib = { }
                rp.Attributes.forEach(a => {
                    attrib[a[0]] = a[1]
                })

                // convert the cidr to an object
                obj = this.parse4(attrib.cidr)
                if (obj != null) {
                    if (attrib.policy != null) {
                        obj.policy = attrib.policy
                    }
                    inets.push(obj)
                }
            })

            // sort the prefixes in ascending order
            var sorted = inets.sort((a,b) => {
                if (a.subnet == b.subnet) {
                    return a.plen - b.plen
                }
                else {
                    return a.subnet - b.subnet
                }
            })
            
            // splice in sorted array
            this.p4.splice(0, this.p4.length, ...sorted)

            // update the free list
            this.updateFree4()

            this.updatePrefixLen({ target: { value: this.filter } })
            
            // all done
            this.state = 'ready'
        },

        // update ipv4 stats
        stats4(subnet, plen) {
            // check DN42 range 172.20.0.0/14
            var masked = (subnet & 0xFFFC0000)>>>0
            if (masked == 0xAC140000) {
                // add em up
                this.stats.nets += 1
                this.stats.addr += 2**(32-plen)
            }
        },

        // recursively check a subnet for free blocks
        scanSubnets(subnet, mask, plen) {

            // check for the last defined prefix
            if (this.ix >= this.p4.length) {
                this.stats4(subnet, plen)
                this.free4.push({
                    subnet: subnet,
                    mask: mask,
                    plen: plen
                })
                return
            }
            
            var prefix = this.p4[this.ix]
            
            // does the next prefix match this subnet ?
            if ((prefix.subnet == subnet) && (prefix.plen == plen)) {
                var policy = 'assigned'
                if (prefix.policy != null) {
                    policy = prefix.policy
                }
                
                if (policy != 'open') {
                    // optimisation to reduce recursion by
                    // scanning forward for open children
                    this.ix += 1
                    
                    while(this.ix < this.p4.length) {
                        prefix = this.p4[this.ix]
                        var masked = (prefix.subnet & mask)>>>0                        
                        if (subnet != masked) {
                            // no longer a child, we're done
                            return
                        }

                        // found a subnet that is open
                        if (prefix.policy == 'open') {
                            this.scanSubnets(prefix.subnet,
                                             prefix.mask,
                                             prefix.plen,
                                             prefix.policy)
                        }
                        else {
                            // don't recurse closed subnets
                            var tmppolicy = 'assigned'
                            if (prefix.policy != null) {
                                tmppolicy = prefix.policy
                            }
                            this.ix += 1
                        }
                    }
                    return
                }

                // move on to next index
                this.ix += 1
                if (this.ix >= this.p4.length) {
                    this.stats4(subnet, plen)
                    this.free4.push({
                        subnet: subnet,
                        mask: mask,
                        plen: plen
                    })
                    return
                }
                prefix = this.p4[this.ix]
                
            }

            // is the next prefix a subnet of the current search ?
            var masked = (prefix.subnet & mask)>>>0
            if (subnet == masked) {
                
                // split the subnet and try again
                plen += 1
                var bit = (2**(32 - plen))>>>0
                mask = (mask | bit)>>>0

                this.scanSubnets((subnet & (~bit))>>>0, mask, plen)
                this.scanSubnets((subnet | bit)>>>0, mask, plen)
            }
            else {
                // found an open block
                this.stats4(subnet, plen)
                this.free4.push({
                    subnet: subnet,
                    mask: mask,
                    plen: plen
                })
            }
        },

        // update the free blocks list
        updateFree4() {
            // reset stats
            this.stats.alloc = this.p4.length
            this.stats.addr = 0
            this.stats.nets = 0            

            // recursively scan for free nets
            this.ix = 0
            this.scanSubnets(0, 0, 0, 'undefined')
        },

        // filter subnets based on prefix length
        filterFree() {

            var tlist = [ ]
            // filter the free list
            this.free4.forEach(free => {
                if (free.plen == this.filter) {
                    tlist.push(free)
                }
            })

            this.ftotal = tlist.length

            // pick up to ten random prefixes
            var result = [ ]
            for(var i = 0; ((tlist.length > 0) && (i < 10)); i++) {
                var ix = Math.round(Math.random() * tlist.length)
                var obj = tlist[ix]

                // push to result
                var octets = [ ]
                octets[0] = (obj.subnet >> 24) & 0xFF
                octets[1] = (obj.subnet >> 16) & 0xFF
                octets[2] = (obj.subnet >> 8) & 0xFF
                octets[3] = obj.subnet & 0xFF
                result.push(octets.join(".")+'/'+obj.plen)
                
                // remove from the list
                tlist.splice(ix, 1)
            }

            this.filtered.splice(0, this.filtered.length, ...result)
        },


        // update function when selecting prefixes
        updatePrefixLen(e) {
            this.filter = e.target.value

            // update buttons
            var group = document.getElementById("prefixselect")
            group.childNodes.forEach(button => {
                if (button.nodeName == "BUTTON") {
                    button.classList.remove('btn-primary')
                    button.classList.remove('btn-secondary')
                    if (button.value == this.filter) {
                        button.classList.add('btn-primary')
                    }
                    else {
                        button.classList.add('btn-secondary')
                    }
                }
            })

            // select available prefixes
            this.filterFree()
            
        },

        // update the example templates
        updateExample(e) {
            this.example = e.text
        }
        
    },
    mounted() {
        // reload data if required
        if (this.p4.length == 0) {
            this.reload()
        }

        // always update the prefix selection
        this.updatePrefixLen({ target: { value: this.filter } })
    }
    
})

//////////////////////////////////////////////////////////////////////////
// free IPv6 view

Vue.component('app-free6', {
    template: '#app-free6-template',
    data() {
        return {

            state: 'invalid',
            error: '',
            inet6num: null,
            p6: [ ],
            stats: {
                alloc: 0,
                nets: 0
            },
            
            plist: [ ]
        }
    },

    methods: {
        
        // reload prefix data
        reload() {
            // reset current data
            this.inet6num = null
            this.p6.splice(0,this.p6.length)
            this.state = "loading"


            // IPv6 prefixes
            axios
                .get('/api/registry/inet6num/*')
                .then(response => {
                    this.inet6num = response.data
                    this.processIPv6()
                })
                .catch(error => {
                    this.error = error
                    this.state = 'error'
                    console.log(error)                    
                })            
        },

        // parse an IPv6 string in to an object
        parse6(cidr) {
            cidr = cidr.replace('_','/')

            // split prefix and length
            var s1 = cidr.split('/')

            var plen = parseInt(s1[1])
            // ignore networks longer than /64 as they aren't valid
            if (plen <= 64) {

                // split out the quads
                var quads = s1[0].split(':')

                // and canonicalise '::'
                var ix=quads.indexOf('')
                if (ix != -1) {
                    while(quads.length < 8) {
                        quads.splice(ix, 0, '0')
                    }
                }

                // calculate 64bit network part of prefix
                var num = BigInt(0)
                var quad
                for(var i = 0; i < 4; i++) {
                    num *= BigInt(65536)
                    quad = (quads[i] == '' ? 0 : parseInt('0x' + quads[i]))
                    num += BigInt(quad)
                }
                
                var plen = parseInt(s1[1])
                var mask = BigInt(0xFFFFFFFFFFFFFFFF) - ((2n**BigInt(64-plen))-1n)

                // return object
                return {
                    plen: plen,
                    mask: mask,
                    subnet: num
                }                
            }
        },
        
        
        // process IPv6 prefixes
        processIPv6() {

            var netspace = BigInt(0)

            var nets = [ ]
            Object.values(this.inet6num).forEach(rp => {
                // extract attributes
                var attrib = { }
                rp.Attributes.forEach(a => {
                    attrib[a[0]] = a[1]
                })

                // turn in to object
                obj = this.parse6(attrib.cidr)
                if (obj != null) {
                    if (attrib.policy != null) {
                        obj.policy = attrib.policy
                    }
                    // ignore ::/0 and fd00::/8 when calculating netspace
                    if ((obj.plen > 8) && (obj.policy != 'open')) {
                        netspace += 2n**BigInt(64 - obj.plen)
                    }
                    nets.push(obj)
                }
            })

            netspace /= 1000000n
            this.stats.nets = Number(netspace)
            
            // sort by subnet
            var sorted = nets.sort((a,b) => {
                if (a.subnet == b.subnet) {
                    return a.plen - b.plen
                }
                else {
                    if (a.subnet > b.subnet) { return 1 }
                    return -1
                }
            })            

            // update
            this.p6.splice(0, this.p6.length, ...nets)

            this.stats.alloc = this.p6.length

            this.generate()

            // all done
            this.state = 'ready'
        },

        // check if a network is already allocated
        check6(n) {

            // fixed /48 mask
            var mask = BigInt(0xFFFFFFFFFFFF0000n)

            var match = null
            this.p6.forEach(obj => {
                // only check longer prefixes than current match
                if ((match == null) || (obj.plen >= match.plen)) {
                    var masked

                    // is the new network a subnet of the test obj ?
                    masked = n & obj.mask
                    if (obj.subnet == masked) {
                        // new, more precise prefix found
                        match = obj
                    }

                    // is the test obj a subnet of this network ?
                    masked = obj.subnet & mask
                    if (n == masked) {
                        // immediate fail, existing subnets not allowed
                        return false
                    }
                }
            })

            if (match.policy != 'open') {
                return false
            }

            return true
        },

        // create new random prefixes
        generate() {

            var nlist = [ ]
            // generate 10 new prefixes
            for(var i = 0; i < 10; i++) {
                var valid = false
                var prefix = ''

                while(!valid) {

                    // create 48bits of random address
                    var quads = [ ]
                    for(var j = 0; j < 3; j++) {
                        quads[j] = Math.round(Math.random()*65536)
                    }
                    // fix the first byte to be in fd00::/8
                    quads[0] = 0xFD00 + (quads[0] & 0xFF)
                    
                    // convert to hex
                    var hex = [ ]
                    for (j = 0; j < 3; j++) {
                        hex[j] = quads[j].toString(16)
                    }
                    
                    prefix = `${hex[0]}:${hex[1]}:${hex[2]}::/48`
                    
                    // convert to a BigInt
                    var num = BigInt(0)
                    for(var j = 0; j < 3; j++) {
                        num *= BigInt(65536)
                        num += BigInt(quads[j])
                    }
                    num *= 65536n

                    // now check that the random prefix is not a subnet of another allocation
                    valid = this.check6(num)
                }
                
                nlist.push(prefix)
            }


            // swap in the new prefix list
            this.plist.splice(0, this.plist.length, ...nlist)
        }
        
    },

    computed: {
        freenets() {
            return Math.round(72057.594 - (this.stats.nets/1000000))
        }
    },

    mounted() {
        if (this.p6.length == 0) {
            this.reload()
        }
    }    
    
})

//////////////////////////////////////////////////////////////////////////
// free IPv6 view

Vue.component('app-asn', {
    template: '#app-asn-template',
    data() {
        return {

            state: 'invalid',
            error: '',
            autnum: null,
            free: [ ],
            stats: {
                alloc: 0,
            },
            
            alist: [ ]
        }
    },

    methods: {
        
        // reload prefix data
        reload() {
            // reset current data
            this.asn = null
            this.free.splice(0,this.free.length)
            this.state = "loading"

            // fetch ASN list
            axios
                .get('/api/registry/aut-num')
                .then(response => {
                    this.autnum = response.data['aut-num']
                    this.processASN()
                })
                .catch(error => {
                    this.error = error
                    this.state = 'error'
                    console.log(error)                    
                })            
        },

        // process ASN list
        processASN() {

            var asn = [ ]
            // extract used ASN
            this.autnum.forEach(a => {
                // only interested in DN42 ASN
                if (a.startsWith('AS424242')) {
                    var num = parseInt(a.substr(8))
                    asn[num] = true
                    this.stats.alloc += 1
                }
            })

            // now work out which ASN are free
            for(var a = 0; a < 4000; a++) {
                if (!asn[a]) {
                    var str = a.toString(10)
                    // zero pad
                    while(str.length < 4) {
                        str = '0' + str
                    }
                    
                    this.free.push('AS424242' + str)
                }
            }

            this.generate()

            // all done
            this.state = 'ready'
        },


        // create new random prefixes
        generate() {

            var nlist = [ ]
            for(var i = 0; i < 10; i++) {
                // pick a random free ASN
                var rand = Math.round(Math.random() * this.free.length)
                nlist.push(this.free[rand])
            }
            
            // swap in the new prefix list
            this.alist.splice(0, this.alist.length, ...nlist)
        }
        
    },

    mounted() {
        if (this.free.length == 0) {
            this.reload()
        }
    }    
    
})

//////////////////////////////////////////////////////////////////////////
// main vue application starts here

// initialise the Vue Router
const router = new VueRouter({
    routes: [
        { path: '/',    component: Vue.component('app-root')  },
        { path: '/4',   component: Vue.component('app-free4') },
        { path: '/6',   component: Vue.component('app-free6') },
        { path: '/asn', component: Vue.component('app-asn')   }
    ]
})

// and the main app instance
const vm = new Vue({
    el: '#free-app',
    data: {
        
    },
    router
})


//////////////////////////////////////////////////////////////////////////
// end of code
