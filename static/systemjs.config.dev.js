(function (global) {
  System.config({
    transpiler: 'ts',
    typescriptOptions: { 
      emitDecoratorMetadata: true,
      experimentalDecorators: true
    },
    paths: {
      // paths serve as alias
      'npm:': '/node_modules/'
    },
    // map tells the System loader where to look for things
    map: {
      // our app is within the app folder
      //'static/ts': '/static/ts',
      // angular bundles
      '@angular/core': 'npm:@angular/core/bundles/core.umd.js',
      '@angular/common': 'npm:@angular/common/bundles/common.umd.js',
      '@angular/compiler': 'npm:@angular/compiler/bundles/compiler.umd.js',
      '@angular/platform-browser': 'npm:@angular/platform-browser/bundles/platform-browser.umd.js',
      '@angular/platform-browser-dynamic': 'npm:@angular/platform-browser-dynamic/bundles/platform-browser-dynamic.umd.js',
      '@angular/http': 'npm:@angular/http/bundles/http.umd.js',
      '@angular/router': 'npm:@angular/router/bundles/router.umd.js',
      //'@angular/forms': 'npm:@angular/forms/bundles/forms.umd.js',
      //'@angular/upgrade': 'npm:@angular/upgrade/bundles/upgrade.umd.js',
      // other libraries
      'rxjs':                      'npm:rxjs',
      'ts': 'npm:plugin-typescript',
      'typescript': 'npm:typescript', 
      //'angular-in-memory-web-api': 'npm:angular-in-memory-web-api/bundles/in-memory-web-api.umd.js'
    },
    // packages tells the System loader how to load when no filename and/or no extension
    packages: {
      "ts": {
        "main": "lib/plugin.js"
      },
      "typescript": {
        "main": "lib/typescript.js",
        "meta": {
          "lib/typescript.js": {
            "exports": "ts"
          }
        }
      },
      rxjs: {
        main: 'index.js',
        defaultExtension: 'js'
      },
      'rxjs/operators' : {
        main: 'index.js',
        defaultExtension: 'js'
      },
      '/static/ts': {
        defaultExtension: 'ts',
        meta: {
          '*.ts': {
            loader: 'ts'
          }
        }
      }
    }
  });
})(this);

