/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  sidebar: [
    'introduction',
    'installation',
    'quick-start',
    'architecture',
    {
      type: 'category',
      label: 'Topics',
      collapsible: true,
      collapsed: false,
      items: [
        'manual-removal',
        'exclusion',
        'customization',
        'metrics'
      ]
    },
    {
      type: 'category',
      label: 'Development',
      collapsible: true,
      collapsed: false,
      items: [
        'setup',
        'releasing',
      ]
    },
    {
      type: 'category',
      label: 'Scanning',
      collapsible: true,
      collapsed: false,
      items: [
        'custom-scanner',
        'trivy',
      ]
    },
    'faq',
    'contributing',
    'code-of-conduct',
  ]
};

module.exports = sidebars;
