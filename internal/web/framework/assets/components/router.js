// AG Router Component  
// Client-side routing (optional)

(function(AG) {
    'use strict';

    AG.router = {
        routes: {},
        
        add: function(path, handler) {
            this.routes[path] = handler;
        },
        
        navigate: function(path) {
            history.pushState(null, '', path);
            this.dispatch(path);
        },
        
        dispatch: function(path) {
            const handler = this.routes[path];
            if (handler) handler();
        }
    };

    console.log('[AG] Router component loaded');

})(window.AG || {});

