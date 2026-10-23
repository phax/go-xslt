import { defineConfig } from 'vitepress'

const repo = 'https://github.com/tim-riep/go-xslt'

export default defineConfig({
  title: 'go-xslt',
  description: 'XSLT 3.0, XPath 3.1 and XSD 1.0 / 1.1 in pure Go',
  // Served as a GitHub Pages project site: https://tim-riep.github.io/go-xslt/
  base: '/go-xslt/',
  cleanUrls: true,
  lastUpdated: true,
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/go-xslt/logo.svg' }],
  ],

  themeConfig: {
    logo: '/logo.svg',

    nav: [
      { text: 'Guide', link: '/guide/getting-started', activeMatch: '/guide/' },
      { text: 'Reference', link: '/reference/engine', activeMatch: '/reference/' },
      { text: 'Desktop tester', link: '/desktop' },
      { text: 'Conformance', link: '/conformance' },
      { text: 'Performance', link: '/performance' },
      {
        text: 'Links',
        items: [
          { text: 'Releases', link: `${repo}/releases` },
          { text: 'pkg.go.dev', link: 'https://pkg.go.dev/github.com/tim-riep/go-xslt' },
        ],
      },
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          items: [
            { text: 'What is go-xslt?', link: '/guide/what-is-go-xslt' },
            { text: 'Getting started', link: '/guide/getting-started' },
            { text: 'Command line', link: '/guide/cli' },
          ],
        },
        {
          text: 'Using the library',
          items: [
            { text: 'Transformations', link: '/guide/transform' },
            { text: 'Compiled stylesheets', link: '/guide/compiled-stylesheets' },
            { text: 'XPath evaluation', link: '/guide/xpath' },
            { text: 'XSD validation', link: '/guide/validation' },
            { text: 'Assertion messages', link: '/guide/assertion-messages' },
            { text: 'Reading a schema', link: '/guide/schema-model' },
            { text: 'Streaming and large documents', link: '/guide/streaming' },
            { text: 'Security', link: '/guide/security' },
            { text: 'Diagnostics', link: '/guide/diagnostics' },
          ],
        },
        {
          text: 'Language support',
          items: [
            { text: 'XSLT 3.0', link: '/guide/xslt' },
            { text: 'XPath 3.1', link: '/guide/xpath-language' },
          ],
        },
      ],
      '/reference/': [
        {
          text: 'API reference',
          items: [
            { text: 'Package engine', link: '/reference/engine' },
            { text: 'Package xsd', link: '/reference/xsd' },
          ],
        },
      ],
    },

    socialLinks: [{ icon: 'github', link: repo }],

    editLink: {
      pattern: `${repo}/edit/main/website/:path`,
      text: 'Edit this page on GitHub',
    },

    search: { provider: 'local' },

    footer: {
      message: 'Released under the Apache License 2.0.',
      copyright: 'Copyright © Tim Riep',
    },
  },
})
