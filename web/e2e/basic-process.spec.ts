import { expect, test } from '@playwright/test';

test('serves the application shell through the generated Go and TypeScript contract', async ({ page, request }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Hello World', exact: true })).toBeVisible();
  await expect(page.getByText('Dex Application', { exact: true })).toBeVisible();
  await expect(page.getByRole('alert')).toHaveCount(0);
  await page.reload();
  await expect(page.getByText('Dex Application', { exact: true })).toBeVisible();
  const health = await request.get('/api/health');
  expect(health.ok()).toBe(true);
  expect(await health.json()).toEqual({ status: 'ok' });
  // The application shell has no parallel Flow management HTTP boundary.
  expect((await request.post('/api/flows', { data: { title: 'Unavailable route' } })).status()).toBe(404);
});
