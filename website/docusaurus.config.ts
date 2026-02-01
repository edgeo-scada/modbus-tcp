import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Edgeo Drivers',
  tagline: 'Librairie Go pour protocoles industriels',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://edgeo.github.io',
  baseUrl: '/drivers/',

  organizationName: 'edgeo',
  projectName: 'drivers',

  onBrokenLinks: 'throw',

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  i18n: {
    defaultLocale: 'fr',
    locales: ['fr'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: '/',
          editUrl: 'https://github.com/edgeo/drivers/tree/main/website/',
          // Versioning
          lastVersion: 'current',
          versions: {
            current: {
              label: '1.0.0',
              badge: true,
            },
          },
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      defaultMode: 'light',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Edgeo Drivers',
      logo: {
        alt: 'Edgeo Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'modbusSidebar',
          position: 'left',
          label: 'Modbus TCP',
        },
        {
          type: 'docsVersionDropdown',
          position: 'right',
          dropdownActiveClassDisabled: true,
        },
        {
          href: 'https://github.com/edgeo/drivers',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'Démarrage rapide',
              to: '/getting-started',
            },
            {
              label: 'Client',
              to: '/client',
            },
            {
              label: 'Serveur',
              to: '/server',
            },
          ],
        },
        {
          title: 'Références',
          items: [
            {
              label: 'Options',
              to: '/options',
            },
            {
              label: 'Erreurs',
              to: '/errors',
            },
            {
              label: 'Métriques',
              to: '/metrics',
            },
          ],
        },
        {
          title: 'Liens',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/edgeo/drivers',
            },
            {
              label: 'Go Package',
              href: 'https://pkg.go.dev/github.com/edgeo/drivers/modbus',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Edgeo. Documentation générée avec Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['go', 'bash', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
