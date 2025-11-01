module.exports = {
  root: true,
  env: {
    browser: true,
    es2021: true,
    node: true,
  },
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-essential',
  ],
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module',
  },
  rules: {
    'vue/no-deprecated-slot-attribute': 'off',
    'vue/multi-word-component-names': 'off',
    'vue/no-mutating-props': 'off',
  },
};
