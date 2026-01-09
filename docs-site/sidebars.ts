import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  guideSidebar: [
    {
      type: 'category',
      label: 'Overview',
      collapsed: false,
      items: [
        'guide/overview/introduction',
        'guide/overview/tech-stack',
        'guide/overview/getting-started',
        'guide/overview/development-schedule',
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      collapsed: false,
      items: [
        'guide/architecture/system-overview',
        'guide/architecture/data-flow',
        'guide/architecture/contracts',
        'guide/architecture/data-requirements',
      ],
    },
    {
      type: 'category',
      label: 'Strategy',
      collapsed: false,
      items: [
        'guide/strategy/stock-selection',
        'guide/strategy/implementation-spec',
        'guide/strategy/verification-checklist',
      ],
    },
    {
      type: 'category',
      label: 'Backend',
      collapsed: true,
      items: [
        'guide/backend/folder-structure',
        'guide/backend/data-layer',
        'guide/backend/signals-layer',
        'guide/backend/selection-layer',
        'guide/backend/portfolio-layer',
        'guide/backend/execution-layer',
        'guide/backend/audit-layer',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      collapsed: false,
      items: [
        'guide/api/overview',
        'guide/api/signals-api',
        'guide/api/selection-api',
        'guide/api/portfolio-api',
        'guide/api/execution-api',
        'guide/api/audit-api',
        'guide/api/errors',
      ],
    },
    {
      type: 'category',
      label: 'Frontend',
      collapsed: true,
      items: [
        'guide/frontend/folder-structure',
      ],
    },
    {
      type: 'category',
      label: 'User Interface',
      collapsed: false,
      items: [
        'guide/ui/design-system',
        'guide/ui/foundation',
        'guide/ui/components',
      ],
    },
    {
      type: 'category',
      label: 'Database',
      collapsed: true,
      items: [
        'guide/database/schema-design',
      ],
    },
    {
      type: 'category',
      label: 'Migration',
      collapsed: true,
      items: [
        'guide/migration/v10-to-v13',
      ],
    },
  ],
};

export default sidebars;
