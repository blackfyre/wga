import * as esbuild from 'esbuild'

await esbuild.build({
    entryPoints: ['src/js/app.js'],
    bundle: true,
    bundle: true,
    minify: true,
    sourcemap: true,
    legalComments: 'linked',
    allowOverwrite: true,
    outdir: '../../assets/public/js/',
})