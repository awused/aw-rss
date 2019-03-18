// https://gist.github.com/aldo-roman/2c437b872b4550bd3f224fec2eaaebb1
const brotli = require('brotli')
const fs = require('fs')
const path = require('path')

const brotliSettings = {
  extension: 'br',
  skipLarger: true,
  mode: 1,      // 0 = generic, 1 = text, 2 = font (WOFF2)
  quality: 10,  // 0 - 11,
  lgwin: 12     // default
};

// Could be done async
const walk = (dir) => fs.readdirSync(dir).forEach(file => {
  const fp = path.join(dir, file)

  const stat = fs.statSync(fp);
  if (stat.isDirectory()) {
    if (fp != 'dist/assets') {
      walk(fp);
    }
    return;
  }

  const result = brotli.compress(fs.readFileSync(fp), brotliSettings)
  fs.writeFileSync(fp + '.br', result)
});

walk('dist');
