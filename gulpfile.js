var gulp = require('gulp');
var tsc = require('gulp-typescript');
var tsProject = tsc.createProject('./tsconfig.json', {
  outFile: 'bundle.js'
});
var embedTemplates = require('gulp-angular-embed-templates');
var uglifyjs = require('uglify-es');
//var uglify = require('gulp-uglify');
var composer = require('gulp-uglify/composer');
var sourcemaps = require('gulp-sourcemaps');
var sass = require('gulp-sass');
var concat = require('gulp-concat');
var clean = require('gulp-clean');
var runSequence = require('run-sequence');
var merge = require('merge-stream');
var change = require('gulp-change');

var minify = composer(uglifyjs, console);

function createErrorHandler(name) {
  return function (err) {
    console.error('Error from ' + name, err.toString());
  };
}

gulp.task('js', function() {
  return merge(
      gulp.src([
            './node_modules/core-js/client/shim.min.js',
            './node_modules/zone.js/dist/zone.js',
            './node_modules/reflect-metadata/Reflect.js',
            './node_modules/systemjs/dist/system.src.js',
            './static/systemjs.config.js'
          ])
          .pipe(concat('system.js'))
          .pipe(minify({
            compress: true
          })),
      /*gulp.src([
            './node_modules/rxjs//',
            './node_modules/@angular/core/bundles/core.umd.min.js',
            './node_modules/@angular/common/bundles/common.umd.min.js',
            './node_modules/@angular/compiler/bundles/compiler.umd.min.js',
            './node_modules/@angular/platform-browser/bundles/platform-browser.umd.min.js',
            './node_modules/@angular/platform-browser-dynamic/bundles/platform-browser-dynamic.umd.min.js',
            './node_modules/@angular/http/bundles/http.umd.min.js',
            './node_modules/@angular/router/bundles/router.umd.min.js'
          ]),*/
      gulp.src(['./static/ts/**/*.ts', './static/js/**/*.js'])
          .pipe(sourcemaps.init())
          .pipe(embedTemplates({
            basePath: '.'
          }))
          .pipe(tsProject()).js
          .pipe(minify({
            compress: true,
            mangle: {
              properties: {
                regex: /_$/
              }
            }
          }))
          .on('error', createErrorHandler('minify-ts'))
          .pipe(sourcemaps.write('/maps'))
      ) // merge()
      .pipe(gulp.dest('./static/compiled/'));
});

gulp.task('js:watch', function() {
  return gulp.watch(['./static/ts/**/*.ts', './static/js/**/*.js', './static/sw.js', './static/templates/**/*.html'], ['js', 'service-worker']);
});

gulp.task('sass', function() {
  return gulp.src('./static/scss/**/*')
      .pipe(sourcemaps.init())
      .pipe(sass({
        outputStyle: 'compressed'
      }).on('error', sass.logError))
      .pipe(concat('main.css'))
      .pipe(sourcemaps.write())
      .pipe(gulp.dest('./static/compiled/'));
});

gulp.task('sass:watch', function () {
  return gulp.watch('./static/scss/**/*', ['sass', 'service-worker']);
});

gulp.task('watch', function() {
  gulp.watch('./static/scss/**/*', ['sass', 'service-worker']);
  gulp.watch(['./static/ts/**/*.ts', './static/js/**/*.js', './static/sw.js', './static/templates/**/*.html'], ['js', 'service-worker']);
});


function changeServiceWorker(content) {
  return content.replace('{CACHE_VERSION}', new Date().getTime())
}

gulp.task('service-worker', function() {
  return gulp.src('./static/sw.js')
      .pipe(change(changeServiceWorker))
      .pipe(gulp.dest('./static/compiled/'));
});

gulp.task('clean', function() {
  return gulp.src('./static/compiled/', {read: false})
    .pipe(clean());
});

gulp.task('default', function(callback) {
  return runSequence('clean',
      ['js', 'sass', 'service-worker'],
      callback);
});
