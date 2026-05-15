import { getRequestConfig } from 'next-intl/server';
import { routing } from './routing';

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
import zhVerifyEmail from '../../messages/zh/verifyEmail.json';
import zhApplications from '../../messages/zh/applications.json';
import zhAuditLogs from '../../messages/zh/auditLogs.json';
import zhPermissions from '../../messages/zh/permissions.json';
import zhRoles from '../../messages/zh/roles.json';

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
import enVerifyEmail from '../../messages/en/verifyEmail.json';
import enApplications from '../../messages/en/applications.json';
import enAuditLogs from '../../messages/en/auditLogs.json';
import enPermissions from '../../messages/en/permissions.json';
import enRoles from '../../messages/en/roles.json';

const zh = { common: zhCommon, errors: zhErrors, auth: zhAuth, validation: zhValidation, profile: zhProfile, dashboard: zhDashboard, users: zhUsers, passwordStrength: zhPasswordStrength, clearData: zhClearData, metadata: zhMetadata, verifyEmail: zhVerifyEmail, applications: zhApplications, auditLogs: zhAuditLogs, permissions: zhPermissions, roles: zhRoles };
const en = { common: enCommon, errors: enErrors, auth: enAuth, validation: enValidation, profile: enProfile, dashboard: enDashboard, users: enUsers, passwordStrength: enPasswordStrength, clearData: enClearData, metadata: enMetadata, verifyEmail: enVerifyEmail, applications: enApplications, auditLogs: enAuditLogs, permissions: enPermissions, roles: enRoles };

const translations = { zh, en } as const;

export default getRequestConfig(async ({ requestLocale }) => {
  let locale = await requestLocale;
  if (!locale || !routing.locales.includes(locale as any)) {
    locale = routing.defaultLocale;
  }
  return {
    locale,
    messages: translations[locale as keyof typeof translations] || translations.zh,
  };
});
