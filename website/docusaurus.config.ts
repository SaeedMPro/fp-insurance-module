import { themes as prismThemes } from 'prism-react-renderer'
import type { Config } from '@docusaurus/types'
import type * as Preset from '@docusaurus/preset-classic'

// The site wraps the repository's own docs/ directory rather than keeping a
// second copy: `docs.path` points at ../docs, so every hand-written page has
// exactly one source of truth and stays readable as plain Markdown on GitHub.
//
// The API reference is different — it is *generated* from
// backend/api/openapi.yaml at build time, so it lands in website/api-generated/
// (gitignored) as its own docs instance. That keeps docs/ entirely hand-written.
const config: Config = {
  title: 'Supplementary Insurance Module',
  tagline: 'Employee top-up health insurance, where benefit policy is data rather than code',
  favicon: 'img/favicon.ico',

  // Lowercase: GitHub serves Pages from the lowercased account name
  // (saeedmpro.github.io), and this value ends up in canonical tags, og:url and
  // the sitemap, so it has to match the origin actually served. The repo segment
  // in baseUrl *is* case-sensitive, and the repo is lowercase.
  url: 'https://saeedmpro.github.io',
  baseUrl: '/fp-insurance-module/',
  organizationName: 'SaeedMPro',
  projectName: 'fp-insurance-module',
  trailingSlash: false,

  // A broken internal link is the failure mode documentation sites actually
  // have, so make it fail the build rather than warn.
  onBrokenLinks: 'throw',
  onBrokenAnchors: 'throw',
  onBrokenMarkdownLinks: 'throw',

  future: { v4: true, faster: true },

  i18n: { defaultLocale: 'en', locales: ['en'] },

  markdown: {
    mermaid: true,
    // .md is parsed as CommonMark, .mdx as MDX. Without this, MDX reads the
    // `{id}` in a path like GET /claims/{id} as a JS expression and the build
    // dies with "id is not defined". It also keeps docs/ as plain Markdown that
    // renders correctly when browsed on GitHub.
    format: 'detect',
  },

  presets: [
    [
      'classic',
      {
        docs: {
          path: '../docs',
          routeBasePath: '/',
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/SaeedMPro/fp-insurance-module/tree/main/',
          // Required by docusaurus-theme-openapi-docs so generated operation
          // pages render with the request/response panels.
          docItemComponent: '@theme/ApiItem',
        },
        blog: false,
        theme: { customCss: './src/css/custom.css' },
      } satisfies Preset.Options,
    ],
  ],

  plugins: [
    // Second docs instance: the generated API reference.
    [
      '@docusaurus/plugin-content-docs',
      {
        id: 'api',
        path: 'api-generated',
        routeBasePath: 'api',
        sidebarPath: './sidebars-api.ts',
        docItemComponent: '@theme/ApiItem',
      },
    ],
    [
      'docusaurus-plugin-openapi-docs',
      {
        id: 'openapi',
        docsPluginId: 'api',
        config: {
          insurance: {
            specPath: '../backend/api/openapi.yaml',
            outputDir: 'api-generated',
            sidebarOptions: { groupPathsBy: 'tag', categoryLinkSource: 'tag' },
          },
        },
      },
    ],
  ],

  themes: ['@docusaurus/theme-mermaid', 'docusaurus-theme-openapi-docs'],

  themeConfig: {
    colorMode: { defaultMode: 'light', respectPrefersColorScheme: true },
    navbar: {
      title: 'Insurance Module',
      items: [
        { type: 'docSidebar', sidebarId: 'docs', position: 'left', label: 'Documentation' },
        { to: '/api', label: 'API', position: 'left' },
        { href: 'https://github.com/SaeedMPro/fp-insurance-module', label: 'GitHub', position: 'right' },
      ],
    },
    footer: {
      style: 'light',
      links: [
        {
          title: 'Documentation',
          items: [
            { label: 'Overview', to: '/' },
            { label: 'Run it locally', to: '/start/run-locally' },
            { label: 'API reference', to: '/api' },
          ],
        },
        {
          title: 'Engineering',
          items: [
            { label: 'Architecture', to: '/engineering/architecture' },
            { label: 'Design decisions', to: '/engineering/decisions' },
            { label: 'Testing strategy', to: '/engineering/testing' },
          ],
        },
        {
          title: 'Source',
          items: [
            { label: 'Repository', href: 'https://github.com/SaeedMPro/fp-insurance-module' },
            { label: 'OpenAPI document', href: 'https://github.com/SaeedMPro/fp-insurance-module/blob/main/backend/api/openapi.yaml' },
          ],
        },
      ],
      copyright: 'Supplementary Insurance Module — bachelor’s capstone project, Bu-Ali Sina University.',
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'go', 'sql', 'json', 'yaml', 'tsx'],
    },
  } satisfies Preset.ThemeConfig,
}

export default config
