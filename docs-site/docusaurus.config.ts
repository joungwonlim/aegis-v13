import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Aegis v13',
  tagline: '기관급 퀀트 트레이딩 시스템',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://aegis-v13.github.io',
  baseUrl: '/',

  organizationName: 'aegis',
  projectName: 'aegis-v13',

  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'ko',
    locales: ['ko'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: 'docs',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/aegis-social-card.jpg',
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Aegis v13',
      logo: {
        alt: 'Aegis Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'guideSidebar',
          position: 'left',
          label: 'Guide',
        },
        {
          href: 'https://github.com/aegis/aegis-v13',
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
              label: 'Getting Started',
              to: '/docs/guide/overview/getting-started',
            },
            {
              label: 'Architecture',
              to: '/docs/guide/architecture/system-overview',
            },
          ],
        },
        {
          title: 'Development',
          items: [
            {
              label: 'Backend',
              to: '/docs/guide/backend/folder-structure',
            },
            {
              label: 'Frontend',
              to: '/docs/guide/frontend/folder-structure',
            },
          ],
        },
        {
          title: 'More',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/aegis/aegis-v13',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Aegis. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'go', 'sql', 'yaml', 'typescript'],
    },
    tableOfContents: {
      minHeadingLevel: 2,
      maxHeadingLevel: 4,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
