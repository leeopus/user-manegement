import { getRequestConfig } from 'next-intl/server';
import { routing } from './routing';

// Import modular translations for Chinese (zh)
import zhCommon from '../../messages/zh/common.json';
import zhErrors from '../../messages/zh/errors.json';
import zhAuth from '../../messages/zh/auth.json';
import zhValidation from '../../messages/zh/validation.json';
import zhProfile from '../../messages/zh/profile.json';
import zhDashboard from '../../messages/zh/dashboard.json';
import zhUsers from '../../messages/zh/users.json';
import zhPasswordStrength from '../../messages/zh/passwordStrength.json';
import zhClearData from '../../messages/zh/clearData.json';
import zhMetadata from '../../messages/zh/metadata.json';

// Import modular translations for English (en)
import enCommon from '../../messages/en/common.json';
import enErrors from '../../messages/en/errors.json';
import enAuth from '../../messages/en/auth.json';
import enValidation from '../../messages/en/validation.json';
import enProfile from '../../messages/en/profile.json';
import enDashboard from '../../messages/en/dashboard.json';
import enUsers from '../../messages/en/users.json';
import enPasswordStrength from '../../messages/en/passwordStrength.json';
import enClearData from '../../messages/en/clearData.json';
import enMetadata from '../../messages/en/metadata.json';

// Merge modular translations for each locale
const translations = {
  zh: {
    common: zhCommon,
    errors: zhErrors,
    auth: zhAuth,
    validation: zhValidation,
    profile: zhProfile,
    dashboard: zhDashboard,
    users: zhUsers,
    passwordStrength: zhPasswordStrength,
    clearData: zhClearData,
    metadata: zhMetadata,
  },
  en: {
    common: enCommon,
    errors: enErrors,
    auth: enAuth,
    validation: enValidation,
    profile: enProfile,
    dashboard: enDashboard,
    users: enUsers,
    passwordStrength: enPasswordStrength,
    clearData: enClearData,
    metadata: enMetadata,
  },
};

export default getRequestConfig(async ({ requestLocale }) => {
  // This typically corresponds to the `[locale]` segment
  let locale = await requestLocale;

  // Ensure that a valid locale is used
  if (!locale || !routing.locales.includes(locale as any)) {
    locale = routing.defaultLocale;
  }

  return {
    locale,
    messages: translations[locale as keyof typeof translations] || translations.zh
  };
});
