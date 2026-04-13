# Internationalization (i18n) Implementation Guide

## Overview

This document describes the internationalization (i18n) implementation for the DevOps Management System frontend. The system now supports multiple languages with easy switching between Chinese (zh-CN) and English (en-US).

## Implementation Details

### 1. Dependencies

- **vue-i18n**: Version 9.x - Official i18n plugin for Vue 3
- Installed via: `npm install vue-i18n@9`

### 2. File Structure

```
devops/web/src/
├── locales/
│   ├── index.ts          # i18n configuration and setup
│   ├── zh-CN.ts          # Chinese language pack
│   └── en-US.ts          # English language pack
├── components/
│   └── LanguageSwitcher.vue  # Language switcher component
├── layouts/
│   └── MainLayout.vue    # Updated to use i18n
└── main.ts               # i18n plugin registration
```

### 3. Language Files

#### Chinese (zh-CN.ts)
Contains all Chinese translations organized by category:
- `menu.*` - Menu item translations
- `breadcrumb.*` - Breadcrumb navigation translations
- `search.*` - Search interface translations
- `common.*` - Common UI element translations

#### English (en-US.ts)
Contains corresponding English translations with the same structure.

### 4. Key Features

#### 4.1 Menu Internationalization
All menu items now use translation keys instead of hardcoded text:

**Before:**
```typescript
{
  key: '/dashboard',
  title: '仪表盘',
  path: '/dashboard'
}
```

**After:**
```typescript
{
  key: '/dashboard',
  titleKey: 'menu.dashboard',  // i18n key
  path: '/dashboard'
}
```

#### 4.2 Breadcrumb Internationalization
Breadcrumbs are dynamically translated based on the current language:

```typescript
const breadcrumbs = computed(() => {
  // ...
  result.push({ title: t('breadcrumb.home'), path: '/dashboard' })
  // ...
})
```

#### 4.3 Search Internationalization
Global search interface supports multiple languages:
- Search placeholder text
- Search result categories
- Quick navigation labels
- UI feedback messages

#### 4.4 Language Switcher Component
A dropdown component in the header allows users to switch languages:
- Shows current language
- Lists available languages
- Persists selection to localStorage
- Updates all UI text immediately

### 5. Usage Examples

#### In Vue Components (Composition API)

```vue
<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

// Use in template
</script>

<template>
  <div>{{ t('menu.dashboard') }}</div>
  <button>{{ t('common.logout') }}</button>
</template>
```

#### Programmatic Language Switching

```typescript
import { setLocale } from '@/locales'

// Switch to English
setLocale('en-US')

// Switch to Chinese
setLocale('zh-CN')
```

#### Getting Current Language

```typescript
import { getLocale } from '@/locales'

const currentLang = getLocale() // Returns 'zh-CN' or 'en-US'
```

### 6. Translation Keys Reference

#### Menu Keys
- `menu.dashboard` - Dashboard
- `menu.app` - Application Management
- `menu.pipeline` - CI/CD Pipeline
- `menu.pipelineDesigner` - Pipeline Designer
- `menu.templates` - Template Market
- `menu.buildCache` - Build Cache
- `menu.buildStats` - Build Statistics
- `menu.quota` - Resource Quota
- `menu.credentials` - Credentials Management
- `menu.variables` - Variables Management
- `menu.healthcheck` - Health Check
- `menu.serviceHealth` - Service Health Check
- `menu.sslCert` - SSL Certificate Check
- `menu.featureFlags` - Feature Flags
- `menu.systemMonitor` - System Monitor
- `menu.resilience` - Resilience Engineering

#### Common Keys
- `common.home` - Home
- `common.profile` - Profile
- `common.logout` - Logout
- `common.search` - Global Search
- `common.select` - Select
- `common.open` - Open
- `common.close` - Close

#### Search Keys
- `search.placeholder` - Search placeholder text
- `search.searching` - Searching indicator
- `search.noResults` - No results message
- `search.quickNav` - Quick navigation title

### 7. Adding New Translations

To add a new translation:

1. **Add to Chinese file** (`zh-CN.ts`):
```typescript
export default {
  menu: {
    // ... existing keys
    newFeature: '新功能',
  }
}
```

2. **Add to English file** (`en-US.ts`):
```typescript
export default {
  menu: {
    // ... existing keys
    newFeature: 'New Feature',
  }
}
```

3. **Use in component**:
```vue
<template>
  <div>{{ t('menu.newFeature') }}</div>
</template>
```

### 8. Adding New Languages

To add support for a new language (e.g., Japanese):

1. Create new language file: `src/locales/ja-JP.ts`
2. Add translations following the same structure
3. Update `src/locales/index.ts`:

```typescript
import jaJP from './ja-JP'

export const SUPPORT_LOCALES = ['zh-CN', 'en-US', 'ja-JP'] as const

export const LOCALE_NAMES: Record<SupportLocale, string> = {
  'zh-CN': '简体中文',
  'en-US': 'English',
  'ja-JP': '日本語',
}

const i18n = createI18n({
  // ...
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
    'ja-JP': jaJP,
  },
})
```

### 9. Best Practices

1. **Always use translation keys**: Never hardcode text in templates
2. **Organize keys logically**: Group related translations together
3. **Keep keys consistent**: Use the same structure across all language files
4. **Test both languages**: Verify translations work correctly in all supported languages
5. **Use descriptive keys**: Make key names self-explanatory (e.g., `menu.pipelineDesigner` not `menu.pd`)

### 10. Language Persistence

The selected language is automatically saved to `localStorage` with the key `locale`. When the user returns to the application, their language preference is restored.

### 11. Fallback Behavior

If a translation key is missing in the selected language, the system falls back to Chinese (zh-CN) as the default language.

### 12. Performance Considerations

- All translations are loaded at application startup
- No additional network requests for language switching
- Language switching is instant with no page reload required
- Menu and breadcrumb translations are computed reactively

### 13. Testing

To verify i18n implementation:

1. **Visual Testing**:
   - Click the language switcher in the header
   - Verify all menu items change language
   - Check breadcrumbs update correctly
   - Test global search interface

2. **Manual Testing Checklist**:
   - [ ] All menu items display in Chinese
   - [ ] All menu items display in English
   - [ ] Language switcher shows current language
   - [ ] Language selection persists after page reload
   - [ ] Breadcrumbs update when language changes
   - [ ] Search interface translates correctly
   - [ ] No hardcoded Chinese text remains

### 14. Troubleshooting

**Issue**: Translations not showing
- **Solution**: Check that the translation key exists in both language files

**Issue**: Language not persisting
- **Solution**: Check browser localStorage is enabled

**Issue**: Some text still in Chinese after switching to English
- **Solution**: Find the hardcoded text and replace with `t('key')`

### 15. Future Enhancements

Potential improvements for the i18n system:
- Add more languages (Japanese, Korean, etc.)
- Implement lazy loading for language files
- Add date/time formatting per locale
- Add number formatting per locale
- Implement pluralization rules
- Add RTL (Right-to-Left) language support

## Conclusion

The i18n implementation provides a solid foundation for multi-language support in the DevOps Management System. All menu items, breadcrumbs, and search interface now support both Chinese and English, with easy extensibility for additional languages.
