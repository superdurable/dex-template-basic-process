import { execFileSync } from 'node:child_process';
import { expect, test } from '@playwright/test';

test('runs approval automation through a durable reminder', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Automation request').fill('Approve the release');
  await page.getByRole('button', { name: 'Start process' }).click();
  const processPanel = page.locator('[data-flow-id]');
  await expect(processPanel).toBeVisible();
  const flowId = await processPanel.getAttribute('data-flow-id');
  expect(flowId).toBeTruthy();
  await expect.poll(() => {
    try {
      execFileSync('dexcli', ['flow', 'skip-timer', flowId!, '-server', process.env.DEX_FLOW_SERVICE_ADDRESS!, '-step-type', 'process.WaitForApproval', '-condition-id', 'approval-reminder', '-yes']);
      return true;
    } catch { return false; }
  }, { timeout: 20_000 }).toBe(true);
  await expect(page.getByTestId('reminder-count')).toHaveText('1');
  await page.getByRole('button', { name: 'Approve' }).click();
  await expect(page.getByTestId('result')).toHaveText('approved automation completed');
});
