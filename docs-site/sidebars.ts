import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */
const sidebars: SidebarsConfig = {
  doclensSidebar: [
    'overview',
    {
      type: 'category',
      label: 'Architecture',
      items: ['architecture/system-context', 'architecture/communication'],
    },
    {
      type: 'category',
      label: 'Services',
      items: [
        'services/catalog',
        'services/api-gateway',
        'services/identity',
        'services/document-intake',
      ],
    },
    {
      type: 'category',
      label: 'Contracts',
      items: ['contracts/overview', 'contracts/events'],
    },
    {
      type: 'category',
      label: 'Operations',
      items: ['operations/development', 'operations/reliability'],
    },
    {
      type: 'category',
      label: 'Architecture decisions',
      items: ['adr/README'],
    },
  ],
};

export default sidebars;
