import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  modbusSidebar: [
    {
      type: 'doc',
      id: 'index',
      label: 'Introduction',
    },
    {
      type: 'doc',
      id: 'getting-started',
      label: 'Démarrage rapide',
    },
    {
      type: 'category',
      label: 'Guide',
      collapsed: false,
      items: [
        'client',
        'server',
        'pool',
      ],
    },
    {
      type: 'category',
      label: 'Référence',
      collapsed: false,
      items: [
        'options',
        'errors',
        'metrics',
      ],
    },
    {
      type: 'category',
      label: 'Exemples',
      items: [
        'examples/basic-client',
        'examples/basic-server',
      ],
    },
    {
      type: 'doc',
      id: 'changelog',
      label: 'Changelog',
    },
  ],
};

export default sidebars;
