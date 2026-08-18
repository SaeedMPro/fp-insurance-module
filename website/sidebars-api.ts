// The generated API reference sidebar. docusaurus-plugin-openapi-docs writes
// api-generated/sidebar.ts on `gen-api-docs` and default-exports the *item
// array* (not a full config), so it gets wrapped in a named sidebar here.
// Both files are build artifacts — neither is hand-edited.
import type { SidebarsConfig } from '@docusaurus/plugin-content-docs'
import items from './api-generated/sidebar'

const sidebars: SidebarsConfig = { openApiSidebar: items }
export default sidebars
