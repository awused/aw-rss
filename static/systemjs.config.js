System.config({
  paths: {
    // paths serve as alias
    'npm:': '/node_modules/'
  },
  // map tells the System loader where to look for things
  map: {
    // our app is within the app folder
    //'module': 'static/compiled/bundle.js',
    // angular bundles
    '@angular/core': 'npm:@angular/core/bundles/core.umd.min.js',
    '@angular/common': 'npm:@angular/common/bundles/common.umd.min.js',
    '@angular/compiler': 'npm:@angular/compiler/bundles/compiler.umd.min.js',
    '@angular/platform-browser': 'npm:@angular/platform-browser/bundles/platform-browser.umd.min.js',
    '@angular/platform-browser-dynamic': 'npm:@angular/platform-browser-dynamic/bundles/platform-browser-dynamic.umd.min.js',
    '@angular/http': 'npm:@angular/http/bundles/http.umd.min.js',
    '@angular/router': 'npm:@angular/router/bundles/router.umd.min.js',
    //'@angular/forms': 'npm:@angular/forms/bundles/forms.umd.js',
    //'@angular/upgrade': 'npm:@angular/upgrade/bundles/upgrade.umd.js',
    // other libraries
    'rxjs':                      'npm:rxjs',
    //'moment':                    'npm:moment',
    //'angular-in-memory-web-api': 'npm:angular-in-memory-web-api/bundles/in-memory-web-api.umd.js'
  },
  // packages tells the System loader how to load when no filename and/or no extension
  packages: {
    rxjs: {
      main: 'index.js',
      defaultExtension: 'js'
    },
    'rxjs/operators' : {
      main: 'index.js',
      defaultExtension: 'js'
    }
  }
});

