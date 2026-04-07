import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

const adminUser = process.env.E2E_ADMIN_USER || 'superadmin'
const adminPassword = process.env.E2E_ADMIN_PASSWORD || 'dD_B7ChB57ZzlV=2'

async function login(page: Page) {
  await page.goto('/login')
  await page.getByPlaceholder('请输入用户名').fill(adminUser)
  await page.getByPlaceholder('请输入密码').fill(adminPassword)
  await page.getByRole('button', { name: '登录后台' }).click()
}

test('login and reach dashboard', async ({ page }) => {
  await login(page)
  await expect(page.getByText('运营看板')).toBeVisible()
})

test('navigate to contests and tracks', async ({ page }) => {
  await login(page)

  await page.getByRole('menuitem', { name: '赛事管理' }).click()
  await expect(page.getByRole('heading', { name: '赛事管理' })).toBeVisible()

  await page.getByRole('menuitem', { name: '赛道管理' }).click()
  await expect(page.getByRole('heading', { name: '赛道管理' })).toBeVisible()
})
