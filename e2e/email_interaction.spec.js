'use strict';

const { test, expect } = require('@playwright/test');
const { InboxPage }    = require('./pages/InboxPage');
const { takeAndAttachScreenshot, screenshotLocator } = require('./helpers/screenshot');

test.describe('Email Interaction Tests', () => {
  /** @type {InboxPage} */
  let inbox;

  test.beforeEach(async ({ page }) => {
    inbox = new InboxPage(page);
    await inbox.goto();
  });

  // ── Read-only tests (run first — don't mutate server state) ─────────────

  test('Basic Email Listing', async () => {
    await expect(inbox.emailList.tbody).toBeVisible();
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });
    expect(await inbox.emailList.count()).toBeGreaterThan(0);
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/interaction-email-listing.png');
  });

  test('View an Email and Interact with Body/Attachments', async () => {
    const firstRow = inbox.emailList.firstRow();
    await expect(firstRow.locator).toBeVisible({ timeout: 10000 });

    await firstRow.open(); // clicks row + waits for initial /body/ response

    await expect(inbox.emailView.locator).toBeVisible();
    await expect(inbox.emailView.backButton).toBeVisible();
    await expect(inbox.emailView.header).toBeVisible();
    await expect(inbox.emailView.content).toBeVisible();

    // Click the first non-active body tab (active tab has no click handler)
    await expect(inbox.emailView.bodyVersions.tabs().first()).toBeVisible();
    const clickedTab = await inbox.emailView.bodyVersions.clickNonActiveTab();
    if (clickedTab) {
      await expect(clickedTab).toHaveClass(/active/);
    }

    // Check attachments section if this email has a paperclip icon
    if (await firstRow.hasAttachment()) {
      await expect(inbox.emailView.attachmentsList).toBeVisible();
      expect(await inbox.emailView.attachmentLinks.count()).toBeGreaterThan(0);
    }

    await screenshotLocator(inbox.emailView.locator, test.info(), 'screenshots/interaction-view-email.png');

    await inbox.emailView.goBack();
    await expect(inbox.emailList.tbody).toBeVisible();
    await expect(inbox.emailView.locator).toBeHidden();
  });

  test('Search/Filter Emails', async () => {
    // Subject search
    const specialCharSubject = 'Test! @#$%^&*()_+ Special Characters';
    await inbox.search.search(specialCharSubject);

    await expect(inbox.emailList.rows().first()).toBeVisible();
    await expect(inbox.emailList.firstRow().previewCell).toContainText(specialCharSubject);
    const subjectCount = await inbox.emailList.count();
    expect(subjectCount).toBeGreaterThan(0);
    await expect(inbox.emailList.pagination.totalMatchesEl).toHaveText(subjectCount.toString());

    // Sender search — the from-cell shows the display name, not the raw address,
    // so we only assert that the search returns results.
    await inbox.search.search('sender@example.com');
    await expect(inbox.emailList.rows().first()).toBeVisible();
    const senderCount = await inbox.emailList.count();
    expect(senderCount).toBeGreaterThan(0);
    await expect(inbox.emailList.pagination.totalMatchesEl).toHaveText(senderCount.toString());

    // No-results search
    await inbox.search.search('zzxxccv_non_existent_query_string');
    await expect(inbox.emailList.emptyMessage()).toBeVisible();
    await expect(inbox.emailList.pagination.totalMatchesEl).toHaveText('0');

    // Clear restores the full list
    await inbox.search.clear();
    expect(await inbox.emailList.count()).toBeGreaterThan(1);
    await expect(inbox.emailList.pagination.totalMatchesEl).not.toHaveText('0');
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/interaction-search-filter.png');
  });

  test('Pagination (requires >20 emails — testdata has 22)', async () => {
    const totalPages  = await inbox.emailList.pagination.totalPages();
    const totalEmails = await inbox.emailList.pagination.totalEmails();

    expect(await inbox.emailList.pagination.currentPage()).toBe(1);

    if (totalPages <= 1) {
      console.warn(`Only ${totalEmails} emails loaded (page size 20) — pagination not testable. Skipping.`);
      await expect(inbox.emailList.pagination.prevButton).toBeDisabled();
      return;
    }

    // Page 1: prev disabled, next enabled
    await expect(inbox.emailList.pagination.prevButton).toBeDisabled();
    await expect(inbox.emailList.pagination.nextButton).toBeEnabled();

    await inbox.emailList.pagination.nextPage();
    await expect(inbox.emailList.pagination.pageStartEl).toHaveText('2');
    await expect(inbox.emailList.pagination.prevButton).toBeEnabled();

    await inbox.emailList.pagination.prevPage();
    await expect(inbox.emailList.pagination.pageStartEl).toHaveText('1');
    await expect(inbox.emailList.pagination.prevButton).toBeDisabled();

    // Navigate to the last page — next should become disabled there
    await inbox.emailList.pagination.goToLastPage();
    await expect(inbox.emailList.pagination.pageStartEl).toHaveText(totalPages.toString());
    await expect(inbox.emailList.pagination.nextButton).toBeDisabled();
    await expect(inbox.emailList.pagination.prevButton).toBeEnabled();
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/interaction-pagination-last-page.png');
  });

  test('Release an Email Modal Interaction', async () => {
    // Open modal from email list row
    await expect(inbox.emailList.firstRow().releaseButton).toBeVisible({ timeout: 10000 });
    await inbox.emailList.firstRow().openReleaseModal();

    await expect(inbox.releaseModal.locator).toBeVisible();
    await expect(inbox.releaseModal.emailIdInput).toBeVisible();
    await expect(inbox.releaseModal.relayConfigSelect).toBeVisible();
    await expect(inbox.releaseModal.senderOriginalRadio).toBeVisible();
    await expect(inbox.releaseModal.overrideSenderInput).toBeVisible();
    await expect(inbox.releaseModal.receiversOriginalRadio).toBeVisible();
    await expect(inbox.releaseModal.overrideReceiversInput).toBeVisible();
    await expect(inbox.releaseModal.releaseButton).toBeVisible();

    await inbox.releaseModal.close();
    await expect(inbox.releaseModal.locator).toBeHidden();

    // Open modal from email-view header button (app bug was fixed in script.js)
    await inbox.emailList.firstRow().open();
    await expect(inbox.emailView.locator).toBeVisible();

    await inbox.emailView.openReleaseModal();
    await expect(inbox.releaseModal.locator).toBeVisible();

    await screenshotLocator(inbox.releaseModal.locator, test.info(), 'screenshots/interaction-release-modal.png');
    await inbox.releaseModal.closeX();
    await expect(inbox.releaseModal.locator).toBeHidden();
  });

  test('Display External Images Toggle', async () => {
    await inbox.search.search('Subject: Email with External Image');

    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 5000 });
    await inbox.emailList.firstRow().open();

    await expect(inbox.emailView.locator).toBeVisible();
    await expect(inbox.emailView.externalImagesToggle).toBeVisible();

    await expect(inbox.emailView.externalImagesToggle).not.toBeChecked();
    await inbox.emailView.externalImagesToggle.check();
    await expect(inbox.emailView.externalImagesToggle).toBeChecked();
    await screenshotLocator(inbox.emailView.locator, test.info(), 'screenshots/interaction-external-images-toggle.png');
    await inbox.emailView.externalImagesToggle.uncheck();
    await expect(inbox.emailView.externalImagesToggle).not.toBeChecked();
  });

  // ── Bulk operations ────────────────────────────────────────────────────

  test('Bulk select — checkbox selects emails and shows toolbar', async () => {
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });

    // The select-all checkbox should be present
    const selectAll = inbox.page.locator('[data-testid="select-all-checkbox"]');
    await expect(selectAll).toBeVisible();

    // Click the first email's checkbox
    const firstCheckbox = inbox.page.locator('[data-testid^="email-checkbox-"]').first();
    await firstCheckbox.check();

    // Bulk toolbar should appear
    const bulkToolbar = inbox.page.locator('#bulk-toolbar');
    await expect(bulkToolbar).toBeVisible();
    await expect(inbox.page.locator('#bulk-count')).toContainText('1 selected');

    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/interaction-bulk-select.png');

    // Uncheck — toolbar should hide
    await firstCheckbox.uncheck();
    await expect(bulkToolbar).not.toBeVisible({ timeout: 5000 });
  });

  test('Bulk select-all — selects all visible emails', async () => {
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });

    const selectAll = inbox.page.locator('[data-testid="select-all-checkbox"]');
    await selectAll.check();

    const visibleCount = await inbox.emailList.count();
    await expect(inbox.page.locator('#bulk-count')).toContainText(visibleCount + ' selected');

    // Deselect all
    await selectAll.uncheck();
    await expect(inbox.page.locator('#bulk-toolbar')).toBeHidden();
  });

  // ── Destructive tests (run last — mutate server state) ──────────────────

  test('Bulk delete — removes selected emails', async () => {
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });
    const initialTotal = await inbox.emailList.pagination.totalEmails();

    // Select first two emails
    const checkboxes = inbox.page.locator('[data-testid^="email-checkbox-"]');
    await checkboxes.nth(0).check();
    await checkboxes.nth(1).check();

    // Click bulk delete — accept the confirmation dialog
    inbox.page.on('dialog', dialog => dialog.accept());
    const resp = inbox.page.waitForResponse(r => r.url().includes('/bulk-delete'));
    await inbox.page.locator('[data-testid="bulk-delete-button"]').click();
    await resp;

    // Should have 2 fewer emails
    await expect(inbox.emailList.pagination.totalMatchesEl)
      .toHaveText((initialTotal - 2).toString(), { timeout: 5000 });

    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/interaction-bulk-delete.png');
  });

  test('Delete a Single Email', async () => {
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });

    // Use total-matches (global count) not visible row count —
    // pagination refills page 1 from page 2 after a deletion.
    const initialTotal = await inbox.emailList.pagination.totalEmails();
    expect(initialTotal).toBeGreaterThan(0);

    await inbox.emailList.firstRow().delete();

    if (initialTotal === 1) {
      await expect(inbox.emailList.emptyMessage()).toBeVisible();
      await expect(inbox.emailList.pagination.totalMatchesEl).toHaveText('0');
    } else {
      await expect(inbox.emailList.pagination.totalMatchesEl)
        .toHaveText((initialTotal - 1).toString());
    }
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/interaction-delete-single.png');
  });

  test('Delete All Emails', async () => {
    await expect(inbox.emailList.rows().first()).toBeVisible({ timeout: 10000 });

    const initialTotal = await inbox.emailList.pagination.totalEmails();
    if (initialTotal === 0) {
      console.warn('No emails to delete. Test might not be effective.');
      return;
    }

    await inbox.emailList.deleteAll();

    await expect(inbox.emailList.emptyMessage()).toBeVisible({ timeout: 5000 });
    await expect(inbox.emailList.rows()).toHaveCount(0);
    await takeAndAttachScreenshot(inbox.page, test.info(), 'screenshots/interaction-delete-all.png');
  });

});
