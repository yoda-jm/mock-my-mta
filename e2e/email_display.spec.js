'use strict';

const { test, expect } = require('@playwright/test');
const { InboxPage }    = require('./pages/InboxPage');
const { takeAndAttachScreenshot, screenshotLocator } = require('./helpers/screenshot');

/**
 * Display and rendering tests — verifies that email content is correctly
 * decoded and shown in the UI.  All tests are read-only (no mutations).
 *
 * Runs first (alphabetical: _display < _features < _interaction).
 */
test.describe('Email Display Tests', () => {
  /** @type {InboxPage} */
  let inbox;

  test.beforeEach(async ({ page }) => {
    inbox = new InboxPage(page);
    await inbox.goto();
  });

  // ── Header fields ────────────────────────────────────────────────────────

  test('Email header shows ID, To and CC fields', async () => {
    // email_various_headers.eml has Cc, display-name From, and explicit To
    await inbox.search.search('subject:Email with Various Headers');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();

    const header = inbox.emailView.header;
    await expect(header).toContainText('ID:');
    await expect(header).toContainText('To:');
    await expect(header).toContainText('CC:');
    await expect(header).toContainText('cc-recipient@example.com');
    await screenshotLocator(header, test.info(), 'screenshots/display-header-fields.png');
  });

  test('From display name shown in header with email as tooltip', async () => {
    // email_various_headers.eml has From: "Sender Name" <sender@example.com>
    await inbox.search.search('subject:Email with Various Headers');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();

    // Header From paragraph should contain the display name
    await expect(inbox.emailView.header).toContainText('Sender Name');

    // The span wrapping the display name carries the raw address as a title
    const fromSpan = inbox.emailView.header.locator('span[title="sender@example.com"]');
    await expect(fromSpan).toBeVisible();
    await expect(fromSpan).toHaveText('Sender Name');
    await screenshotLocator(inbox.emailView.header, test.info(), 'screenshots/display-from-display-name.png');
  });

  test('From raw address shown in list row when no display name', async () => {
    // email_unique_from.eml has From: uniquesender@filter-test.net (no display name)
    await inbox.search.search('from:uniquesender@filter-test.net');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    // The from cell should contain the raw address directly (no display name)
    const firstRow = inbox.emailList.firstRow();
    await expect(firstRow.fromCell).toContainText('uniquesender@filter-test.net');
    await screenshotLocator(firstRow.locator, test.info(), 'screenshots/display-from-raw-address.png');
  });

  // ── Decoding ─────────────────────────────────────────────────────────────

  test('Quoted-printable body decoded correctly — non-ASCII chars visible', async () => {
    await inbox.search.search('subject:Quoted-Printable French Characters');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();

    // The plain-text tab is selected by default for this plain-text-only email
    // Decoded QP should show the actual UTF-8 characters, not escape sequences
    const bodyText = await inbox.page.evaluate(() => {
      const host = document.querySelector('.email-content');
      return host?.shadowRoot?.textContent ?? '';
    });

    expect(bodyText).toContain('élève');
    expect(bodyText).toContain('café');
    expect(bodyText).toContain('français');
    expect(bodyText).not.toContain('=C3=');
    await screenshotLocator(inbox.emailView.locator, test.info(), 'screenshots/display-quoted-printable.png');
  });

  test('Base64 encoded body decoded correctly', async () => {
    await inbox.search.search('subject:Base64 Encoded Body');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();

    const bodyText = await inbox.page.evaluate(() => {
      const host = document.querySelector('.email-content');
      return host?.shadowRoot?.textContent ?? '';
    });

    expect(bodyText).toContain('Hello World! This body is base64 encoded.');
    await screenshotLocator(inbox.emailView.locator, test.info(), 'screenshots/display-base64-decoded.png');
  });

  test('CID images are rewritten to API endpoint in rendered HTML body', async () => {
    await inbox.search.search('subject:Test CID Image Only');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();

    // Wait for body to render then inspect the shadow DOM image src
    // The server rewrites src="cid:myimage123" → src="/api/emails/{id}/cid/myimage123"
    const imgSrc = await inbox.page.evaluate(() => {
      const host = document.querySelector('.email-content');
      const img = host?.shadowRoot?.querySelector('img');
      return img?.getAttribute('src') ?? null;
    });

    expect(imgSrc).toMatch(/^\/api\/emails\/.+\/cid\//);
    await screenshotLocator(inbox.emailView.locator, test.info(), 'screenshots/display-cid-image-rewrite.png');
  });

  test('CID image request is served by the API (200 response)', async ({ page }) => {
    const cidInbox = new InboxPage(page);
    await cidInbox.goto();

    await cidInbox.search.search('subject:Test CID Image Only');
    await expect(cidInbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    // Intercept the /cid/ API request that the browser makes when rendering the img
    const cidRequest = page.waitForResponse(r => r.url().includes('/cid/'));
    await cidInbox.emailList.firstRow().open();
    const resp = await cidRequest;

    expect(resp.status()).toBe(200);
    expect(resp.headers()['content-type']).toMatch(/^image\//);
    await screenshotLocator(cidInbox.emailView.locator, test.info(), 'screenshots/display-cid-image-served.png');
  });

  // ── External images ───────────────────────────────────────────────────────

  test('External images are hidden by default and shown when toggled on', async () => {
    await inbox.search.search('subject:Test External Image Only');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();

    // Toggle starts unchecked — external images must be hidden
    await expect(inbox.emailView.externalImagesToggle).not.toBeChecked();

    const externalImgDisplayOff = await inbox.page.evaluate(() => {
      const host = document.querySelector('.email-content');
      const img = host?.shadowRoot?.querySelector('img[src^="http"]');
      return img?.style.display ?? null;
    });
    expect(externalImgDisplayOff).toBe('none');

    // Check the toggle — images should become visible
    await inbox.emailView.externalImagesToggle.check();
    await expect(inbox.emailView.externalImagesToggle).toBeChecked();

    const externalImgDisplayOn = await inbox.page.evaluate(() => {
      const host = document.querySelector('.email-content');
      const img = host?.shadowRoot?.querySelector('img[src^="http"]');
      return img?.style.display ?? null;
    });
    expect(externalImgDisplayOn).toBe('');
    await screenshotLocator(inbox.emailView.locator, test.info(), 'screenshots/display-external-images-toggled.png');
  });

  // ── Release modal: receiver override ─────────────────────────────────────

  test('Release modal — receiver override enables the input field', async () => {
    await expect(inbox.emailList.firstRow().releaseButton).toBeVisible({ timeout: 10000 });
    await inbox.emailList.firstRow().openReleaseModal();

    await expect(inbox.releaseModal.locator).toBeVisible();

    // By default the override receivers input is disabled
    await expect(inbox.releaseModal.overrideReceiversInput).toBeDisabled();

    // Clicking the override radio enables it
    const overrideRadio = inbox.releaseModal.locator.locator(
      '[data-testid="release-modal-receivers-override-radio"]'
    );
    await overrideRadio.click();
    await expect(inbox.releaseModal.overrideReceiversInput).toBeEnabled();

    // Fill a value and verify
    await inbox.releaseModal.overrideReceiversInput.fill('custom@test.com');
    await expect(inbox.releaseModal.overrideReceiversInput).toHaveValue('custom@test.com');

    // Switch back to original — input becomes disabled
    await inbox.releaseModal.receiversOriginalRadio.click();
    await expect(inbox.releaseModal.overrideReceiversInput).toBeDisabled();

    await screenshotLocator(inbox.releaseModal.locator, test.info(), 'screenshots/display-release-receiver-override.png');
    await inbox.releaseModal.close();
    await expect(inbox.releaseModal.locator).toBeHidden();
  });

  // ── Search autocomplete ───────────────────────────────────────────────────

  test('Search autocomplete — typing partial filter key shows suggestion', async () => {
    const suggestionDisplay = inbox.page.locator('#suggestion-display');

    // Wait for the suggestions API response triggered by typing "sub"
    const resp = inbox.page.waitForResponse(r =>
      r.url().includes('/api/filters/suggestions') && r.url().includes('term=sub')
    );
    await inbox.search.input.fill('sub');
    // Trigger keyup so the JS fires
    await inbox.search.input.dispatchEvent('keyup');
    await resp;

    // The suggestion display should show "subject:<text>"
    await expect(suggestionDisplay).toHaveText(/subject:/);
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/display-autocomplete-suggestion.png');
  });

  test('Search autocomplete — Tab key completes the suggestion', async () => {
    const suggestionDisplay = inbox.page.locator('#suggestion-display');

    const resp = inbox.page.waitForResponse(r =>
      r.url().includes('/api/filters/suggestions') && r.url().includes('term=sub')
    );
    await inbox.search.input.fill('sub');
    await inbox.search.input.dispatchEvent('keyup');
    await resp;

    await expect(suggestionDisplay).toHaveText(/subject:/);

    // Tab should complete the input to the full suggestion
    await inbox.search.input.press('Tab');
    await expect(inbox.search.input).toHaveValue(/subject:/);
    await expect(suggestionDisplay).toHaveText('');
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/display-autocomplete-tab-complete.png');
  });
});
