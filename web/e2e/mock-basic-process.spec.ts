import { expect, test } from '@playwright/test';

test.skip(process.env.E2E_MOCK !== 'true', 'requires the local mock server');

test('exercises the complete mock UI lifecycle and failures', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByText('Mock controls')).toBeVisible();

  await page.getByRole('button', { name: 'Fail next Start' }).click();
  await page.getByRole('button', { name: 'Start process' }).click();
  await expect(page.getByRole('alert')).toContainText('mock start failure');

  await page.getByRole('button', { name: 'Start process' }).click();
  const processPanel = page.locator('[data-flow-id]');
  await expect(processPanel).toBeVisible();
  await expect(processPanel.locator('.status')).toHaveText('waiting for approval');

  await page.reload();
  await expect(processPanel).toBeVisible();
  await expect(processPanel.locator('.status')).toHaveText('waiting for approval');

  await page.getByRole('button', { name: 'Fail next Refresh' }).click();
  await expect(page.getByRole('alert')).toContainText('mock refresh failure');
  await page.getByRole('button', { name: 'Retry' }).click();
  await expect(page.getByText('mock refresh failure')).not.toBeVisible();

  await page.getByRole('button', { name: 'Emit reminder' }).click();
  await expect(page.getByTestId('reminder-count')).toHaveText('1');

  await page.getByRole('button', { name: 'Fail next Approval' }).click();
  await page.getByRole('button', { name: 'Approve' }).click();
  await expect(page.getByRole('alert')).toContainText('mock approval failure');
  await page.getByRole('button', { name: 'Approve' }).click();
  await expect(page.getByTestId('result')).toHaveText('approved automation completed');

  await page.getByRole('button', { name: 'Reset' }).click();
  await expect(processPanel).not.toBeVisible();
  await expect(page.getByRole('button', { name: 'Fail next Start' })).toBeVisible();
});
