//////////////////////////////////////////////////////////////////////////
// DN42 Registry Explorer
//////////////////////////////////////////////////////////////////////////

// global store for data that is loaded once
const GlobalStore = {
    data: {
        RegStats: null,
        Index: null
    }
}


//////////////////////////////////////////////////////////////////////////
// registry object component

Vue.component('reg-object', {
    template: '#reg-object-template',
    props: [ 'link', 'content' ],
    data() {
        return { }
    },
    methods: {
        updateSearch: function(str) {
            vm.updateSearch(str)
        }
    },
    computed: {
        rtype: function() {
            var ix = this.link.indexOf("/")
            return this.link.substring(0, ix)
        },
        obj: function() {
            var ix = this.link.indexOf("/")
            return this.link.substring(ix + 1)            
        }
    }
})

//////////////////////////////////////////////////////////////////////////
// reg-attribute component

Vue.component('reg-attribute', {
    template: '#reg-attribute-template',
    props: [ 'content' ],
    data() {
        return { }
    },
    methods: {
        isRegObject: function(str) {
            return (this.content.match(/^\[.*?\]\(.*?\)/) != null)
        }
    },
    computed: {
        objectLink: function() {
            reg = this.content.match(/^\[(.*?)\]\((.*?)\)/)
            return reg[2]
        },
        decorated: function() {
            var c = this.content

            // an attribute terminated with \n indicates a blank
            // trailing line, however a single trailing <br/> will
            // not be rendered in HTML, this hack doubles up the
            // trailing newline so it creates a <br/> pair
            // which renders as the blank line
            if (c.substr(c.length-1) == "\n") {
                c = c + "\n"
            }

            // replace newlines with line breaks
            c = c.replace(/\n/g, "<br/>")

            // decorate
            c = anchorme(c, {
                truncate: 40,
                ips: false,
                attributes: [ { name: "target", value: "_blank" } ]
            })

            // and return the final decorated content
            return c
        }
    }
})

//////////////////////////////////////////////////////////////////////////
// search input component

Vue.component('search-input', {
    template: '#search-input-template',
    data() {
        return {
            search: '',
            searchTimeout: 0
        }
    },
    methods: {

        debounceSearch: function(value) {
            if (this.searchTimeout) {
                clearTimeout(this.searchTimeout)
            }

            // link should be an absolute path
            value = '/' + value
            
            this.searchTimeout = setTimeout(
                this.$router.push.bind(this.$router, value), 500
            )
        }
    },
    mounted() {

        // listen to search updates and set the search text appropriately
        this.$root.$on('SearchChanged', value => {
            this.search = value
        })
    }
})

//////////////////////////////////////////////////////////////////////////
// registry-stats component

Vue.component('registry-stats', {
    template: '#registry-stats-template',
    data() {
        return {
            state: null,
            error: null,
            store: GlobalStore.data
        }
    },
    methods: {

        // just fetch the stats from the API server via axios
        reload(event) {
            this.store.RegStats = null
            this.state = "loading"

            axios
                .get('/api/registry/')
                .then(response => {
                    this.store.RegStats = response.data
                    this.state = 'complete'
                })
                .catch(error => {
                    this.error = error
                    this.state = 'error'
                    console.log(error)                    
                })
        }
    },
    mounted() {
        if (this.store.RegStats == null) {
            this.reload()
        }
    }
})

//////////////////////////////////////////////////////////////////////////
// root component

Vue.component('app-root', {
    template: '#app-root-template',
    mounted() {
        this.$root.$emit('SearchChanged', '')        
    }
})

//////////////////////////////////////////////////////////////////////////
// search results

Vue.component('app-search', {
    template: '#app-search-template',
    data() {
        return {
            state: null,
            search: null,
            results: null,
            store: GlobalStore.data
        }
    },

    // trigger a search on route update and on mount
    beforeRouteUpdate(to, from, next) {
        this.search = to.params.pathMatch
        this.doSearch()
        next()
    },
    mounted() {
        // store the search for later
        this.search = this.$route.params.pathMatch
        
        // the index must be loaded before any real work takes place
        if (this.store.Index == null) {
            this.loadIndex()
        }
        else {
            // index was already loaded, go search
            this.doSearch()
        }
    },
    
    methods: {

        // load the search index from the API
        loadIndex() {

            this.state = 'loading'
            this.$root.$emit('SearchChanged', 'Initialising ...')
            
            axios
                .get('/api/registry/*')
                .then(response => {
                    this.store.Index = response.data
                    
                    // if a query parameter has been passed,
                    // then go search
                    if (this.search != null) {
                        this.$nextTick(this.doSearch())
                    }
                })
                .catch(error => {
                    this.state = 'error'
                    console.log(error)
                })            
        },

        // substring match object names against a search
        matchObjects(objects, rtype, term) {
            var results = [ ]

            // check each object
            for (const obj in objects) {

                // matches are all lower case
                var s = objects[obj].toLowerCase()
                
                var pos = s.indexOf(term)
                if (pos != -1) {
                    if ((pos == 0) && (s == term)) {
                        // exact match, return just this result
                        return [[ rtype, objects[obj] ]]
                    }
                    results.push([ rtype, objects[obj] ])
                }
            }
            
            return results            
        },
        

        // filter the index to find matches
        searchFilter() {
            
            var results = [ ]
            var index = this.store.Index
            
            // comparisons are lowercase
            var term = this.search.toLowerCase()
            
            // check if search includes a '/'
            var slash = term.indexOf('/')
            if (slash != -1) {
                // match only on the specific type
                var rtype = term.substring(0, slash)
                var term = term.substring(slash + 1)
                objects = index[rtype]
                
                if (objects != null) {
                    results = this.matchObjects(objects, rtype, term)
                }
            }
            else {
                
                // walk though the entire index
                for (const rtype in index) {
                    var objlist = this.matchObjects(index[rtype], rtype, term)
                    results = results.concat(objlist)
                }
            }
            
            return results
            
        },

        // perform the search and present results
        doSearch() {
            // notify other components that the search is updated
            this.$root.$emit('SearchChanged', this.search)

            // filter matches against the index
            filtered = this.searchFilter()

            // got nothing ?
            if (filtered.length == 0) {
                this.state = "noresults"
            }

            // just one result
            else if (filtered.length == 1) {
                var objname = filtered[0]

                // load the object from the API
                this.state = 'loading'
                query = '/api/registry/' + objname[0] + '/' + objname[1]
                
                axios
                    .get(query)
                    .then(response => {
                        this.state = 'result'
                        this.result = response.data
                    })
                    .catch(error => {
                        this.error = error
                        this.state = 'error'
                    })
            }

            // lots of results
            else {
                this.state = "resultlist"
                this.result = filtered
            }
        }
    }
})

//////////////////////////////////////////////////////////////////////////
// main vue application starts here

// initialise the Vue Router
const router = new VueRouter({
    routes: [
        { path: '/',   component: Vue.component('app-root')   },
        { path: '/*',  component: Vue.component('app-search') }
    ]
})

// and the main app instance
const vm = new Vue({
    el: '#explorer_app',
    data: {
        
    },
    router
})


//////////////////////////////////////////////////////////////////////////
// end of code
