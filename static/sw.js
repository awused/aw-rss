var CACHE_VERSION = {CACHE_VERSION};
var CACHE_NAME = 'desu-rss-cache-v' + CACHE_VERSION;
var urlsToCache = [
  '/static/compiled/bundle.js',
  '/static/compiled/system.js',
  '/static/compiled/main.css',
];

self.addEventListener('install', function(event) {
  // Perform install steps
  self.skipWaiting();
  event.waitUntil(
      caches.open(CACHE_NAME).then((cache) => { cache.addAll(urlsToCache); }));
});

shouldCache = (url) =>
    url.startsWith(location.origin) && (url.indexOf('static/compiled') !== -1 ||
                                        url.indexOf('node_modules') !== -1)

self.addEventListener('fetch', function(event) {
  event.respondWith(
      caches.open(CACHE_NAME)
          .then((cache) => cache.match(event.request).then(function(response) {
            if (response) {
              return response;
            }

            // We want to cache everything in node_modules or static/compiled
            return fetch(event.request)
                .then(
                    (response) => {
                      if (response && shouldCache(event.request.url)) {
                        cache.put(event.request, response.clone());
                      }

                      return response;
                    },
                    (error) => {
                      console.error(error);
                      throw error;
                    });
          })));
});

self.addEventListener('activate', function(event) {
  event.waitUntil(clients.claim());
  event.waitUntil(caches.keys().then(function(cacheNames) {
    return Promise.all(cacheNames
                           .filter(function(cacheName) {
                             // Should use sw.js changing as a signal to clear
                             // all caches
                             return CACHE_NAME !== cacheName;
                           })
                           .map(function(cacheName) {
                             console.log("deleted: " + cacheName);
                             return caches.delete(cacheName);
                           }));
  }));
});
