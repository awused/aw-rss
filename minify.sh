#!/bin/sh

cd "$(dirname "$0")"

try_tsc () {
  if hash tsc 2>/dev/null; then
    tsc $@
  else
    ./node_modules/typescript/bin/tsc $@
  fi
}

try_uglify () {
  if hash uglifyjs 2>/dev/null; then
    uglifyjs $@
  else
    ./node_modules/uglify-js/bin/uglifyjs $@
  fi
}

rm static/compiled/*
#rm -r static/*/external/node_modules/*

compass compile -s compressed

try_tsc -outFile static/compiled/intermediate.js
# All angular2.0.0 releases after beta.0 break when running mangled code
# See https://github.com/angular/angular/issues/6380
#try_uglify --compress --screw-ie8 --mangle --mangle-props --mangle-regex="/_$/"
try_uglify --compress --screw-ie8 --mangle-props --mangle-regex="/_$/" \
    --in-source-map static/compiled/intermediate.js.map \
    --source-map static/compiled/main.js.map --source-map-url main.js.map \
    -o static/compiled/main.js -- static/compiled/intermediate.js

rm static/compiled/intermediate*

