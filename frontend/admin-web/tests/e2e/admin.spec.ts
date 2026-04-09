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
  await expect(page.getByRole('heading', { name: '比赛看板' })).toBeVisible()
})

test('navigate to major admin modules', async ({ page }) => {
  await login(page)

  await page.getByRole('menuitem', { name: '作者管理' }).click()
  await expect(page.getByRole('heading', { name: '作者管理' })).toBeVisible()

  await page.getByRole('menuitem', { name: '子管理员' }).click()
  await expect(page.getByRole('heading', { name: '子管理员管理' })).toBeVisible()

  await page.getByRole('menuitem', { name: '比赛看板' }).click()
  await expect(page.getByRole('heading', { name: '比赛看板' })).toBeVisible()

  await page.getByRole('button', { name: '比赛配置入口' }).click()
  await expect(page.getByRole('heading', { name: '赛事管理' })).toBeVisible()

  await page.getByRole('button', { name: '详情' }).first().click()
  await expect(page.getByRole('heading', { name: '比赛详情' })).toBeVisible()

  await page.getByRole('menuitem', { name: '全局配置' }).click()
  await expect(page.getByRole('heading', { name: '全局配置' })).toBeVisible()
})
