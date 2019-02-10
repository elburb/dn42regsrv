//////////////////////////////////////////////////////////////////////////
// DN42 Registry Explorer
//////////////////////////////////////////////////////////////////////////

//////////////////////////////////////////////////////////////////////////
// registry-stats component

Vue.component('registry-stats', {
    template: '#registry-stats-template',
    data() {
        return {
            state: "loading",
            error: "",
            types: null,
        }
    },
    methods: {
        updateSearch: function(str) {
            vm.updateSearch(str)
        },
        
        reload: function(event) {
            this.types = null,
            this.state = "loading"

            axios
                .get('/api/registry/')
                .then(response => {
                    this.types = response.data
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
        this.reload()
    }
})

//////////////////////////////////////////////////////////////////////////
// registry object component

Vue.component('reg-object', {
    template: '#reg-object-template',
    props: [ 'link' ],
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
            return anchorme(this.content.replace(/\n/g, "<br/>"), {
                truncate: 40,
                ips: false,
                attributes: [ { name: "target", value: "_blank" } ]                
            })
        }
    }
})

//////////////////////////////////////////////////////////////////////////
// construct a search URL from a search term

function matchObjects(objects, rtype, term) {

    var results = [ ]
    
    for (const obj in objects) {
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
}


function searchFilter(index, term) {

    var results = [ ]

    // comparisons are lowercase
    term = term.toLowerCase()

    // includes a '/' ? search only in that type
    var slash = term.indexOf('/')
    if (slash != -1) {
        var rtype = term.substring(0, slash)
        var term = term.substring(slash + 1)
        objects = index[rtype]
        if (objects != null) {
            results = matchObjects(objects, rtype, term)
        }
    }
    else {
        // walk though the entire index
        for (const rtype in index) {
            results = results.concat(matchObjects(index[rtype], rtype, term))
        }
    }

    return results
}

//////////////////////////////////////////////////////////////////////////
// main application

// application data
var appData = {
    searchInput: '',
    searchTimeout: 0,
    state: '',
    debug: "",
    index: null,
    filtered: null,
    result: null
}

// methods
var appMethods = {

    loadIndex: function(event) {

        this.state = 'loading'
        this.searchInput = 'Initialisation ...'
        
        axios
            .get('/api/registry/*')
            .then(response => {
                this.index = response.data

                // if a query parameter has been passed,
                // update the search
                if (window.location.search != "") {
                    var param = window.location.search.substr(1)
                    this.$nextTick(this.updateSearch(param))
                }
                else {
                    this.state = ''
                    this.searchInput = ''
                }

            })
            .catch(error => {
                // what to do here ?
                console.log(error)
            })
    },

    // called on every search input change
    debounceSearchInput: function(value) {
        
        if (this.search_timeout) {
            clearTimeout(this.search_timeout)
        }

        // reset if searchbox is empty
        if (value == "") {
            this.state = ""
            this.searchInput = ""
            this.filtered = null
            this.results = null
            return
        }
        
        this.search_timeout =
            setTimeout(this.updateSearch.bind(this,value),500)
    },

    // called after the search input has been debounced
    updateSearch: function(value) {
        this.searchInput = value
        this.filtered = searchFilter(this.index, value)
        if (this.filtered.length == 0) {
            this.state = "noresults"
        }
        else if (this.filtered.length == 1) {
            this.state = "loading"
            var details = this.filtered[0]

            query = '/api/registry/' + details[0] + '/' + details[1]
            
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
        else {
            this.state = "resultlist"
            this.result = this.filtered
        }
    },

    copyPermalink: function() {

        // create a temporary textarea element off the page
        var target = document.createElement("textarea")
        target.style.position = "absolute"
        target.style.left = "-9999px"
        target.style.top = "0"
        target.id = "_hidden_permalink_"
        document.body.appendChild(target)

        // set the text area content
        target.textContent = this.permalink

        // copy it to the clipboard
        var currentFocus = document.activeElement
        target.focus()
        target.select()
        document.execCommand('copy')

        // and return to normal
        if (currentFocus && typeof currentFocus.focus === "function") {
            currentFocus.focus()
        }
        
        document.body.removeChild(target)
    }

}


// intialise Vue instance

var vm = new Vue({
    el: '#explorer',
    data: appData,
    methods: appMethods,
    computed: {
        permalink: function() {
            return window.location.origin + '/?' + this.searchInput
        }
    },
    mounted() {
        this.loadIndex()
        this.$nextTick(function() {
            $('.popover-dismiss').popover({
                trigger: 'focus'
            })
        })
    }
})



//////////////////////////////////////////////////////////////////////////
// end of code
