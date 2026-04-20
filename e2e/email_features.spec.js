'use strict';

const { test, expect } = require('@playwright/test');
const { InboxPage }    = require('./pages/InboxPage');
const { takeAndAttachScreenshot, screenshotLocator } = require('./helpers/screenshot');

/**
 * Feature tests covering UI behaviours not tested in email_interaction.spec.js.
 * This file is intentionally non-destructive (no delete-all) so it can safely
 * run before the interaction suite (alphabetical order: _features < _interaction).
 *
 * The one exception is "Delete from email view" which removes a single email
 * and is placed last in this file.
 */
test.describe('Email Feature Tests', () => {
  /** @type {InboxPage} */
  let inbox;

  test.beforeEach(async ({ page }) => {
    inbox = new InboxPage(page);
    await inbox.goto();
  });

  // ── Read-only ────────────────────────────────────────────────────────────

  test('Refresh button reloads the email list', async () => {
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });
    const countBefore = await inbox.emailList.count();

    await inbox.emailList.refresh();

    // Count should be identical after a plain refresh
    await expect(inbox.emailList.rows().first()).toBeVisible();
    expect(await inbox.emailList.count()).toBe(countBefore);
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-refresh-button.png');
  });

  test('Mailbox sidebar — filter by recipient then reset to all', async () => {
    const totalBefore = await inbox.emailList.pagination.totalEmails();

    // Expand the mailbox list; items are populated after the API call
    await inbox.mailbox.expand();
    await expect(inbox.mailbox.item('recipient1@example.com')).toBeVisible();

    // Click a mailbox — list should show only emails addressed to that recipient
    await inbox.mailbox.clickMailbox('recipient1@example.com');
    const filteredCount = await inbox.emailList.pagination.totalEmails();
    expect(filteredCount).toBeGreaterThan(0);
    expect(filteredCount).toBeLessThan(totalBefore);

    // The search box should reflect the mailbox filter
    await expect(inbox.search.input).toHaveValue('mailbox:recipient1@example.com');
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-mailbox-filter.png');

    // Click "All" to reset
    await inbox.mailbox.showAll();
    const totalAfter = await inbox.emailList.pagination.totalEmails();
    expect(totalAfter).toBe(totalBefore);
  });

  test('Syntax help modal — opens with filter entries and closes', async () => {
    await inbox.syntaxHelp.open();

    await expect(inbox.syntaxHelp.locator).toBeVisible();
    // The table should have at least one filter syntax entry
    expect(await inbox.syntaxHelp.tableRows.count()).toBeGreaterThan(0);
    await screenshotLocator(inbox.syntaxHelp.locator, test.info(), 'screenshots/features-syntax-help-modal.png');

    await inbox.syntaxHelp.close();
    await expect(inbox.syntaxHelp.locator).toBeHidden();
  });

  test('Body version tabs — switch between HTML, plain-text and raw', async () => {
    // The Apple Watch email has html, plain-text, watch-html and raw versions
    await inbox.search.search('subject:Apple Watch Example');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();

    const { bodyVersions } = inbox.emailView;

    // html is the default preferred version — it should already be bold
    await expect(bodyVersions.tab('html')).toBeVisible();
    await expect(bodyVersions.tab('html')).toHaveClass(/active/);

    // Switch to plain-text
    await bodyVersions.switchTo('plain-text');
    await expect(bodyVersions.tab('plain-text')).toHaveClass(/active/);
    await expect(bodyVersions.tab('html')).not.toHaveClass('active');

    // Switch to raw
    await bodyVersions.switchTo('raw');
    await expect(bodyVersions.tab('raw')).toHaveClass(/active/);
    await expect(bodyVersions.tab('plain-text')).not.toHaveClass('active');

    // Switch back to html
    await bodyVersions.switchTo('html');
    await expect(bodyVersions.tab('html')).toHaveClass(/active/);
    await screenshotLocator(inbox.emailView.locator, test.info(), 'screenshots/features-body-version-tabs.png');
  });

  test('Watch-HTML body version tab is present for Apple Watch email', async () => {
    await inbox.search.search('subject:Apple Watch Example');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();

    // watch-html is a non-standard version only present in Apple Watch emails
    const watchTab = inbox.emailView.bodyVersions.tab('watch-html');
    await expect(watchTab).toBeVisible();

    // Clicking it fetches the watch-html body and marks it active
    await inbox.emailView.bodyVersions.switchTo('watch-html');
    await expect(watchTab).toHaveClass(/active/);

    // The watch-html body should render with actual content
    const bodyText = await inbox.page.evaluate(() => {
      const host = document.querySelector('.email-content');
      return host?.shadowRoot?.textContent ?? '';
    });
    expect(bodyText).toContain('Watch HTML part');

    await screenshotLocator(inbox.emailView.locator, test.info(), 'screenshots/features-watch-html-tab.png');
  });

  test('Attachments — list and download links for multipart email', async () => {
    // "Important information" has 3 attachments
    await inbox.search.search('subject:Important information');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    // Verify the paperclip icon is shown in the list row before opening
    const firstRow = inbox.emailList.firstRow();
    await expect(firstRow.attachmentIcon).toBeVisible();

    await firstRow.open();

    await expect(inbox.emailView.locator).toBeVisible();
    await expect(inbox.emailView.attachmentsList).toBeVisible();
    // Attachment links are rendered asynchronously after the /attachments/ call
    await expect(inbox.emailView.attachmentLinks.first()).toBeVisible({ timeout: 5000 });
    expect(await inbox.emailView.attachmentLinks.count()).toBeGreaterThanOrEqual(1);

    // Each link should have a valid href pointing to the attachment content endpoint
    const href = await inbox.emailView.attachmentLinks.first().getAttribute('href');
    expect(href).toMatch(/\/api\/emails\/.+\/attachments\/.+\/content/);
    await screenshotLocator(inbox.emailView.attachmentsList, test.info(), 'screenshots/features-attachments-list.png');
  });

  test('Email header shows From, Date and Subject', async () => {
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();
    await expect(inbox.emailView.header).toBeVisible();

    // The header section should contain all three key fields
    await expect(inbox.emailView.header).toContainText('From:');
    await expect(inbox.emailView.header).toContainText('Date:');
    await expect(inbox.emailView.header).toContainText('Subject:');
    await screenshotLocator(inbox.emailView.header, test.info(), 'screenshots/features-email-header.png');
  });

  test('Release modal — sender override enables the input field', async () => {
    await expect(inbox.emailList.firstRow().releaseButton).toBeVisible({ timeout: 10000 });
    await inbox.emailList.firstRow().openReleaseModal();

    await expect(inbox.releaseModal.locator).toBeVisible();

    // By default the override sender input is disabled
    await expect(inbox.releaseModal.overrideSenderInput).toBeDisabled();

    // Clicking the override radio enables it
    const overrideRadio = inbox.releaseModal.locator.locator(
      '[data-testid="release-modal-sender-override-radio"]'
    );
    await overrideRadio.click();
    await expect(inbox.releaseModal.overrideSenderInput).toBeEnabled();

    // Type a value and verify it's accepted
    await inbox.releaseModal.overrideSenderInput.fill('custom@test.com');
    await expect(inbox.releaseModal.overrideSenderInput).toHaveValue('custom@test.com');

    // Switch back to original — input becomes disabled and cleared
    await inbox.releaseModal.senderOriginalRadio.click();
    await expect(inbox.releaseModal.overrideSenderInput).toBeDisabled();

    await screenshotLocator(inbox.releaseModal.locator, test.info(), 'screenshots/features-release-sender-override.png');
    await inbox.releaseModal.close();
    await expect(inbox.releaseModal.locator).toBeHidden();
  });

  test('Release modal — override values are sent in the relay request', async () => {
    await expect(inbox.emailList.firstRow().releaseButton).toBeVisible({ timeout: 10000 });
    await inbox.emailList.firstRow().openReleaseModal();
    await expect(inbox.releaseModal.locator).toBeVisible();

    // Enable sender override and fill in a custom sender
    const senderOverrideRadio = inbox.releaseModal.locator.locator(
      '[data-testid="release-modal-sender-override-radio"]'
    );
    await senderOverrideRadio.click();
    await inbox.releaseModal.overrideSenderInput.fill('override-sender@test.com');

    // Enable receiver override and fill in a custom receiver
    const receiversOverrideRadio = inbox.releaseModal.locator.locator(
      '[data-testid="release-modal-receivers-override-radio"]'
    );
    await receiversOverrideRadio.click();
    await inbox.releaseModal.overrideReceiversInput.fill('override-receiver@test.com');

    // Intercept the relay POST request to verify the submitted values
    const relayRequest = inbox.page.waitForRequest(r =>
      r.url().includes('/relay') && r.method() === 'POST'
    );
    await inbox.releaseModal.releaseButton.click();
    const req = await relayRequest;
    const body = req.postDataJSON();

    expect(body.sender).toBe('override-sender@test.com');
    expect(body.recipients).toEqual(['override-receiver@test.com']);
  });

  test('Release modal — resets to original on re-open', async () => {
    await expect(inbox.emailList.firstRow().releaseButton).toBeVisible({ timeout: 10000 });

    // First open: toggle overrides
    await inbox.emailList.firstRow().openReleaseModal();
    await expect(inbox.releaseModal.locator).toBeVisible();

    const senderOverrideRadio = inbox.releaseModal.locator.locator(
      '[data-testid="release-modal-sender-override-radio"]'
    );
    await senderOverrideRadio.click();
    await inbox.releaseModal.overrideSenderInput.fill('leftover@test.com');
    await inbox.releaseModal.close();
    await expect(inbox.releaseModal.locator).toBeHidden();

    // Second open: overrides should be reset to original
    await inbox.emailList.firstRow().openReleaseModal();
    await expect(inbox.releaseModal.locator).toBeVisible();

    await expect(inbox.releaseModal.overrideSenderInput).toBeDisabled();
    await expect(inbox.releaseModal.overrideSenderInput).toHaveValue('');
    await expect(inbox.releaseModal.overrideReceiversInput).toBeDisabled();
    await expect(inbox.releaseModal.overrideReceiversInput).toHaveValue('');

    await inbox.releaseModal.close();
  });

  // ── Search filter operators ───────────────────────────────────────────────

  test('has:attachment filter — returns only emails with attachments', async () => {
    const totalBefore = await inbox.emailList.pagination.totalEmails();

    await inbox.search.search('has:attachment');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    const filteredCount = await inbox.emailList.pagination.totalEmails();
    expect(filteredCount).toBeGreaterThan(0);
    expect(filteredCount).toBeLessThan(totalBefore);

    // Every visible row must show the paperclip icon
    const rowCount = await inbox.emailList.count();
    for (let i = 0; i < rowCount; i++) {
      await expect(inbox.emailList.row(i).attachmentIcon).toBeVisible();
    }
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-filter-has-attachment.png');
  });

  test('from: filter — returns only emails from the given sender', async () => {
    // email_unique_from.eml is the only email from uniquesender@filter-test.net
    await inbox.search.search('from:uniquesender@filter-test.net');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    expect(await inbox.emailList.pagination.totalEmails()).toBe(1);
    await expect(inbox.emailList.firstRow().toCell)
      .toContainText('recipient@example.com');
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-filter-from.png');
  });

  test('before: filter — returns only emails before given date', async () => {
    // Combine with subject: to avoid matching zero-date emails (no Date header
    // → time.Time{} which sorts before any real date and would match before:)
    // email_dated_old.eml subject is "Old Email from 2020", dated 2020-06-15
    await inbox.search.search('subject:"Old Email from 2020" before:2021-01-01');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    expect(await inbox.emailList.pagination.totalEmails()).toBe(1);

    // Same email dated 2020-06-15 must NOT appear when cut-off is before that date
    await inbox.search.search('subject:"Old Email from 2020" before:2020-01-01');
    await expect(inbox.emailList.emptyMessage()).toBeVisible({ timeout: 5000 });
    expect(await inbox.emailList.pagination.totalEmails()).toBe(0);
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-filter-before.png');
  });

  test('after: filter — returns only emails after given date', async () => {
    // email_dated_recent.eml is dated 2026-04-01 — only match after 2026-01-01
    await inbox.search.search('subject:"Recent Email from 2026" after:2026-01-01');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    expect(await inbox.emailList.pagination.totalEmails()).toBe(1);

    // Same email should not appear for a far-future after: date
    await inbox.search.search('subject:"Recent Email from 2026" after:2099-12-31');
    await expect(inbox.emailList.emptyMessage()).toBeVisible({ timeout: 5000 });
    expect(await inbox.emailList.pagination.totalEmails()).toBe(0);
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-filter-after.png');
  });

  test('older_than: filter — returns only emails older than the given duration', async () => {
    // All emails except email_dated_recent (2026-04-01) are older than 1 year
    const totalBefore = await inbox.emailList.pagination.totalEmails();

    await inbox.search.search('older_than:1y');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    const filtered = await inbox.emailList.pagination.totalEmails();
    expect(filtered).toBeGreaterThan(0);
    expect(filtered).toBeLessThan(totalBefore);
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-filter-older-than.png');
  });

  test('newer_than: filter — returns only emails newer than the given duration', async () => {
    // email_dated_recent.eml is dated 2026-04-01; newer_than:30d gives a safe margin
    await inbox.search.search('newer_than:30d');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    expect(await inbox.emailList.pagination.totalEmails()).toBe(1);
    await expect(inbox.emailList.firstRow().previewCell)
      .toContainText('Recent Email from 2026');
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-filter-newer-than.png');
  });

  test('Combined filters — from: and has:attachment', async () => {
    // "Important information" email is from no-reply@example.com and has attachments
    await inbox.search.search('from:no-reply@example.com has:attachment');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    const count = await inbox.emailList.pagination.totalEmails();
    expect(count).toBeGreaterThan(0);
    // All results must have the paperclip
    for (let i = 0; i < count; i++) {
      await expect(inbox.emailList.row(i).attachmentIcon).toBeVisible();
    }
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-filter-combined.png');
  });

  test('Quoted phrase search — finds emails matching exact phrase', async () => {
    // email_with_specialchars.eml body contains "special characters"
    await inbox.search.search('"special characters"');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    expect(await inbox.emailList.pagination.totalEmails()).toBeGreaterThan(0);

    // A phrase that exists nowhere should return nothing
    await inbox.search.search('"xyzzy_no_match_phrase_abc"');
    await expect(inbox.emailList.emptyMessage()).toBeVisible({ timeout: 5000 });
    expect(await inbox.emailList.pagination.totalEmails()).toBe(0);
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-filter-quoted-phrase.png');
  });

  // ── URL routing / deep linking ──────────────────────────────────────

  test('URL hash updates when searching', async () => {
    await inbox.search.search('from:sender@example.com');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    // Hash should reflect the search query
    const hash = await inbox.page.evaluate(() => window.location.hash);
    expect(hash).toContain('#/search/');
    expect(hash).toContain('sender');
  });

  test('URL hash updates when opening an email', async () => {
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });
    await inbox.emailList.firstRow().open();
    await expect(inbox.emailView.locator).toBeVisible();

    const hash = await inbox.page.evaluate(() => window.location.hash);
    expect(hash).toMatch(/#\/email\//);
  });

  test('URL hash updates when switching tabs', async () => {
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });
    await inbox.emailList.firstRow().open();
    await expect(inbox.emailView.locator).toBeVisible();

    // Click the headers tab
    const headersTab = inbox.page.locator('[data-testid="email-body-version-tab-headers"]');
    const resp = inbox.page.waitForResponse(r => r.url().includes('/headers'));
    await headersTab.click();
    await resp;

    const hash = await inbox.page.evaluate(() => window.location.hash);
    expect(hash).toContain('/headers');
  });

  test('Deep link to search restores results', async ({ page }) => {
    const deepInbox = new InboxPage(page);
    // Navigate directly to a search URL
    await page.goto('/#/search/from%3Auniquesender%40filter-test.net');
    await page.waitForSelector('[data-testid="email-list-body"]');
    await expect(deepInbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    expect(await deepInbox.emailList.pagination.totalEmails()).toBe(1);
  });

  test('URL normalizes to #/ for empty search on page 1', async () => {
    await inbox.search.clear();
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    const hash = await inbox.page.evaluate(() => window.location.hash);
    expect(hash).toBe('#/');
  });

  // ── Wait-for-email API ─────────────────────────────────────────────────

  test('waitForEmail API — immediate match returns email with URL and count', async () => {
    const result = await inbox.waitForEmail('from:uniquesender@filter-test.net', '5s');
    expect(result.email.id).toBeTruthy();
    expect(result.email.from.address).toBe('uniquesender@filter-test.net');
    expect(result.total_matches).toBe(1);
    expect(result.url).toContain('/#/email/');
  });

  test('waitForEmail API — multiple matches returns count', async () => {
    const result = await inbox.waitForEmail('from:sender@example.com', '5s');
    expect(result.email.id).toBeTruthy();
    expect(result.total_matches).toBeGreaterThan(1);
    expect(result.url).toContain('/#/email/');
  });

  test('waitForEmail API — timeout returns error', async () => {
    let error;
    try {
      await inbox.waitForEmail('from:zzz_never_exists@nowhere.test', '1s');
    } catch (e) {
      error = e;
    }
    expect(error).toBeDefined();
    expect(error.message).toContain('408');
  });

  test('gotoEmail — navigate directly to email by ID', async () => {
    // Get an email ID via the API first
    const result = await inbox.waitForEmail('from:uniquesender@filter-test.net', '5s');
    const emailId = result.email.id;

    // Navigate directly using deep link
    await inbox.gotoEmail(emailId);
    await expect(inbox.emailView.locator).toBeVisible();
    await expect(inbox.emailView.header).toContainText('uniquesender@filter-test.net');
  });

  test('gotoSearch — navigate directly to search results', async () => {
    await inbox.gotoSearch('has:attachment');
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });

    // All visible rows should have the paperclip
    const count = await inbox.emailList.count();
    expect(count).toBeGreaterThan(0);
    for (let i = 0; i < count; i++) {
      await expect(inbox.emailList.row(i).attachmentIcon).toBeVisible();
    }
  });

  // ── Slightly destructive (single delete) — placed last ──────────────────

  test('Delete from email view — removes email and returns to list', async () => {
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });
    const totalBefore = await inbox.emailList.pagination.totalEmails();

    await inbox.emailList.firstRow().open();
    await expect(inbox.emailView.locator).toBeVisible();

    await inbox.emailView.delete();

    // After deletion the app returns to the list view automatically
    await expect(inbox.emailList.tbody).toBeVisible({ timeout: 5000 });
    await expect(inbox.emailView.locator).toBeHidden();
    await expect(inbox.emailList.pagination.totalMatchesEl)
      .toHaveText((totalBefore - 1).toString());
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/features-delete-from-view.png');
  });

});
