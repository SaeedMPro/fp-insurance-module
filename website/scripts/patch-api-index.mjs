// docusaurus-plugin-openapi-docs generates the spec's info page with its own
// slug, which leaves the /api route with no index — every "API reference" link
// then 404s and, with onBrokenLinks: 'throw', fails the build.
//
// Giving that page `slug: /` makes it the index of the api docs instance. Run
// after gen-api-docs (see the `gen-api` npm script) so it survives regeneration.
import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const dir = 'api-generated'
const info = readdirSync(dir).find((f) => f.endsWith('.info.mdx'))
if (!info) {
  console.error('patch-api-index: no .info.mdx found — did gen-api-docs run?')
  process.exit(1)
}
const path = join(dir, info)
let s = readFileSync(path, 'utf8')
if (s.includes('\nslug: /')) {
  console.log(`patch-api-index: ${info} already has slug: /`)
} else {
  s = s.replace(/^(---\nid: .*\n)/m, '$1slug: /\n')
  writeFileSync(path, s)
  console.log(`patch-api-index: added slug: / to ${info}`)
}
