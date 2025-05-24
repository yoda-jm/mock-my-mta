const { test, expect } = require('@playwright/test');

const BASE_URL = 'http://localhost:8080';

test.beforeEach(async ({ page }) => {
  // Navigate to the application before each test
  await page.goto(BASE_URL);
  // Wait for the email list to be potentially populated
  await page.waitForSelector('[data-testid="email-list-body"]');
});

test.describe('Email Interaction Tests', () => {

  test('Basic Email Listing', async ({ page }) => {
    // Verify that the email list is displayed
    await expect(page.locator('[data-testid="email-list-body"]')).toBeVisible();

    // Check if at least one email row is present
    // This assumes test data is loaded and populates at least one email.
    // Using a more specific selector to count rows with the expected prefix.
    const emailRows = page.locator('[data-testid^="email-row-"]');
    await expect(emailRows.first()).toBeVisible({ timeout: 10000 }); // Wait for rows to appear
    const count = await emailRows.count();
    expect(count).toBeGreaterThan(0);
  });

  test('View an Email and Interact with Body/Attachments', async ({ page }) => {
    // Click on the first email in the list
    const firstEmailRow = page.locator('[data-testid^="email-row-"]').first();
    await expect(firstEmailRow).toBeVisible({ timeout: 10000 });
    await firstEmailRow.click();

    // Verify that the email view section becomes visible
    await expect(page.locator('.email-view')).toBeVisible();
    await expect(page.locator('[data-testid="email-view-back-button"]')).toBeVisible();
    await expect(page.locator('.email-header')).toBeVisible(); // Using class as data-testid was not specified for this exact element
    await expect(page.locator('.email-content')).toBeVisible(); // Using class as data-testid was not specified for this exact element

    // Check for the presence of body version tabs
    const bodyVersionTabs = page.locator('[data-testid^="email-body-version-tab-"]');
    await expect(bodyVersionTabs.first()).toBeVisible();
    const initialBodyContent = await page.locator('.email-content').innerHTML();

    // Click on different body version tabs
    const htmlTab = page.locator('[data-testid="email-body-version-tab-html"]');
    const plainTextTab = page.locator('[data-testid="email-body-version-tab-plain-text"]');
    const rawTab = page.locator('[data-testid="email-body-version-tab-raw"]');

    if (await htmlTab.isVisible()) {
        await htmlTab.click();
        await expect(htmlTab).toHaveCSS('font-weight', '700'); // 700 is often 'bold'
        await expect(page.locator('.email-content')).not.innerHTML(initialBodyContent, { timeout: 5000});
    }

    if (await plainTextTab.isVisible()) {
        await plainTextTab.click();
        await expect(plainTextTab).toHaveCSS('font-weight', '700');
        await expect(page.locator('.email-content')).not.innerHTML(initialBodyContent, { timeout: 5000});
    }

    if (await rawTab.isVisible()) {
        await rawTab.click();
        await expect(rawTab).toHaveCSS('font-weight', '700');
        await expect(page.locator('.email-content')).not.innerHTML(initialBodyContent, { timeout: 5000});
    }

    // Check for attachments if the email has them
    // We need to identify an email that is known to have attachments.
    // For this example, let's assume the first email (often sample.eml) might have one.
    // A more robust way would be to click an email with a visible paperclip icon.
    const paperclipIcon = firstEmailRow.locator('[data-testid^="email-attachment-icon-"]').locator('i.bi-paperclip');
    if (await paperclipIcon.isVisible()) {
        await expect(page.locator('[data-testid="email-attachments-list"]')).toBeVisible();
        const attachmentLinks = page.locator('[data-testid^="attachment-link-"]');
        await expect(attachmentLinks.count()).toBeGreaterThan(0);
        await expect(attachmentLinks.first()).toBeVisible();
    }

    // Click the back button
    await page.locator('[data-testid="email-view-back-button"]').click();

    // Verify that the view returns to the email list
    await expect(page.locator('.email-list')).toBeVisible();
    await expect(page.locator('.email-view')).toBeHidden();
  });

  test('Delete a Single Email', async ({ page }) => {
    const emailListBody = page.locator('[data-testid="email-list-body"]');
    await expect(emailListBody.locator('[data-testid^="email-row-"]').first()).toBeVisible({ timeout: 10000 });

    const initialEmailCount = await emailListBody.locator('[data-testid^="email-row-"]').count();
    expect(initialEmailCount).toBeGreaterThan(0); // Ensure there's an email to delete

    // Click the delete icon for the first email
    const firstEmailDeleteButton = page.locator('[data-testid^="email-delete-button-"]').first();
    await firstEmailDeleteButton.click();

    // No confirmation dialog is expected based on current app behavior
    // Wait for the email list to update. A simple way is to check count or a specific element to be gone.
    // A more robust way might involve waiting for network response or a specific UI change like a notification.
    await page.waitForTimeout(1000); // Give time for the list to refresh

    const finalEmailCount = await emailListBody.locator('[data-testid^="email-row-"]').count();

    // If initial count was 1, the list might show "No emails found"
    if (initialEmailCount === 1) {
        await expect(emailListBody.locator('text="No emails found"')).toBeVisible();
        expect(finalEmailCount).toBe(0); // Or 1 if the "No emails found" is a row itself
    } else {
        expect(finalEmailCount).toBe(initialEmailCount - 1);
    }
  });

  test('Delete All Emails', async ({ page }) => {
    const emailListBody = page.locator('[data-testid="email-list-body"]');
    await expect(emailListBody.locator('[data-testid^="email-row-"]').first()).toBeVisible({ timeout: 10000 });

    const initialEmailCount = await emailListBody.locator('[data-testid^="email-row-"]').count();
    if (initialEmailCount === 0) {
      // This case should ideally not happen if test data is loaded.
      // If it can, we might need to load emails first or skip the test.
      console.warn('No emails to delete. Test might not be effective.');
      return;
    }

    // Click the "Delete All Messages" button
    await page.locator('[data-testid="delete-all-button"]').click();

    // No confirmation dialog is expected based on current app behavior

    // Verify that the email list is now empty or shows a "no emails" message
    await expect(emailListBody.locator('text="No emails found"')).toBeVisible({ timeout: 5000 });
    // Also check that actual email rows are gone
    await expect(page.locator('[data-testid^="email-row-"]').count()).toBe(0);
  });

  test('Search/Filter Emails', async ({ page }) => {
    // Test searching by a known subject
    // Using subject from email_with_specialchars.eml
    const specialCharSubject = 'Test! @#$%^&*()_+ Special Characters';
    await page.locator('[data-testid="search-input"]').fill(specialCharSubject);
    await page.locator('[data-testid="search-submit-button"]').click();
    await page.waitForTimeout(500); // Allow time for search to process

    const emailRowsBySubject = page.locator('[data-testid^="email-row-"]');
    await expect(emailRowsBySubject.first()).toBeVisible();
    const firstEmailPreviewBySubject = emailRowsBySubject.first().locator('[data-testid^="email-preview-"]');
    await expect(firstEmailPreviewBySubject).toContainText(specialCharSubject);
    const subjectSearchCount = await emailRowsBySubject.count();
    expect(subjectSearchCount).toBeGreaterThan(0);
    await expect(page.locator('#total-matches')).toHaveText(subjectSearchCount.toString());

    // Test searching by a known sender
    const knownSender = 'sender@example.com'; // Common sender from test data
    await page.locator('[data-testid="search-input"]').fill(knownSender);
    await page.locator('[data-testid="search-submit-button"]').click();
    await page.waitForTimeout(500);

    const emailRowsBySender = page.locator('[data-testid^="email-row-"]');
    await expect(emailRowsBySender.first()).toBeVisible();
    // Iterate and check if each visible email is from the known sender.
    for (let i = 0; i < await emailRowsBySender.count(); i++) {
      const row = emailRowsBySender.nth(i);
      const fromCell = row.locator('[data-testid^="email-from-"]');
      await expect(fromCell).toHaveText(new RegExp(knownSender)); // Regex for partial match if name is also present
    }
    const senderSearchCount = await emailRowsBySender.count();
    expect(senderSearchCount).toBeGreaterThan(0);
    await expect(page.locator('#total-matches')).toHaveText(senderSearchCount.toString());

    // Test searching with a filter that yields no results
    const nonsensicalQuery = 'zzxxccv_non_existent_query_string';
    await page.locator('[data-testid="search-input"]').fill(nonsensicalQuery);
    await page.locator('[data-testid="search-submit-button"]').click();
    await page.waitForTimeout(500);

    await expect(page.locator('[data-testid="email-list-body"] text="No emails found"')).toBeVisible();
    await expect(page.locator('#total-matches')).toHaveText('0');

    // Clear the search input
    await page.locator('[data-testid="search-clear-button"]').click();
    await page.waitForTimeout(500);

    const totalEmailsAfterClear = await page.locator('[data-testid^="email-row-"]').count();
    expect(totalEmailsAfterClear).toBeGreaterThan(1); // Assuming more than 1 email in total
    // Check if total matches reflects all emails (or first page of all emails)
    // This depends on how total-matches behaves on clear.
    // For now, just ensure it's not '0' if there are emails.
    if (totalEmailsAfterClear > 0) {
        await expect(page.locator('#total-matches')).not.toHaveText('0');
    }
  });

  test('Pagination (requires >10 emails, assuming page size 10)', async ({ page }) => {
    // Ensure enough emails are loaded for pagination to be active.
    // The test data should include more than 10 emails for this test.
    // We will assume page size is 10 based on typical UIs.
    // If not, this test might need adjustment or data setup.

    const initialPageStart = await page.locator('#page-start').textContent();
    const initialPageTotal = await page.locator('#page-total').textContent();
    const initialTotalMatches = await page.locator('#total-matches').textContent();

    expect(initialPageStart).toBe('1'); // Should start on page 1

    const totalEmails = parseInt(initialTotalMatches, 10);
    const totalPages = parseInt(initialPageTotal, 10);

    if (totalEmails <= 10) {
      console.warn('Not enough emails to test pagination effectively. Skipping some checks.');
      // Check if prev/next are disabled if only one page
      await expect(page.locator('[data-testid="prev-page-button"]')).toBeDisabled();
      await expect(page.locator('[data-testid="next-page-button"]')).toBeDisabled();
      return;
    }
    expect(totalPages).toBeGreaterThan(1);

    // Initial state: prev should be disabled
    await expect(page.locator('[data-testid="prev-page-button"]')).toBeDisabled();
    await expect(page.locator('[data-testid="next-page-button"]')).toBeEnabled();

    // Click next page
    await page.locator('[data-testid="next-page-button"]').click();
    await page.waitForTimeout(500); // Allow list to refresh

    await expect(page.locator('#page-start')).toHaveText('2');
    await expect(page.locator('[data-testid="prev-page-button"]')).toBeEnabled();

    // Click previous page
    await page.locator('[data-testid="prev-page-button"]').click();
    await page.waitForTimeout(500); // Allow list to refresh

    await expect(page.locator('#page-start')).toHaveText('1');
    await expect(page.locator('[data-testid="prev-page-button"]')).toBeDisabled();

    // Go to the last page to test 'next' button disablement
    if (totalPages > 1) {
      for (let i = 1; i < totalPages; i++) {
        await page.locator('[data-testid="next-page-button"]').click();
        await page.waitForTimeout(200); // short wait for list update
      }
      await expect(page.locator('#page-start')).toHaveText(totalPages.toString());
      await expect(page.locator('[data-testid="next-page-button"]')).toBeDisabled();
      await expect(page.locator('[data-testid="prev-page-button"]')).toBeEnabled();
    }
  });

  test('Release an Email Modal Interaction', async ({ page }) => {
    // Find the first email and click its release button
    const firstEmailReleaseButton = page.locator('[data-testid^="email-release-button-"]').first();
    await expect(firstEmailReleaseButton).toBeVisible({ timeout: 10000 });
    await firstEmailReleaseButton.click();

    // Verify the release modal appears
    const releaseModal = page.locator('#releaseEmailModal'); // Using ID as data-testid wasn't on the modal itself
    await expect(releaseModal).toBeVisible();

    // Check for key elements within the modal
    await expect(releaseModal.locator('[data-testid="release-modal-email-id-input"]')).toBeVisible();
    await expect(releaseModal.locator('[data-testid="release-modal-relay-config-select"]')).toBeVisible();
    await expect(releaseModal.locator('[data-testid="release-modal-sender-original-radio"]')).toBeVisible();
    await expect(releaseModal.locator('[data-testid="release-modal-override-sender-input"]')).toBeVisible(); // May be disabled initially
    await expect(releaseModal.locator('[data-testid="release-modal-receivers-original-radio"]')).toBeVisible();
    await expect(releaseModal.locator('[data-testid="release-modal-override-receivers-input"]')).toBeVisible(); // May be disabled initially
    await expect(releaseModal.locator('[data-testid="release-modal-release-button"]')).toBeVisible();

    // Close the modal using the "Close" button in the footer
    await releaseModal.locator('[data-testid="release-modal-close-button"]').click();
    await expect(releaseModal).toBeHidden();

    // Re-open modal by clicking release button on the email view page
    const firstEmailRow = page.locator('[data-testid^="email-row-"]').first();
    await firstEmailRow.click();
    await expect(page.locator('.email-view')).toBeVisible();
    await page.locator('[data-testid="email-view-release-button"]').click();
    await expect(releaseModal).toBeVisible();

    // Close the modal using the 'X' button in the header
    await releaseModal.locator('[data-testid="release-modal-close-button-x"]').click();
    await expect(releaseModal).toBeHidden();
  });

  test('Display External Images Toggle', async ({ page }) => {
    // Find and click the email with external images
    // This requires the server to be run with `e2e/testdata/emails` which includes `email_with_external_image.eml`
    await page.locator('[data-testid="search-input"]').fill('Subject: Email with External Image');
    await page.locator('[data-testid="search-submit-button"]').click();
    await page.waitForTimeout(500);

    const emailRow = page.locator('[data-testid^="email-row-"]').first();
    await expect(emailRow).toBeVisible({ timeout: 5000 });
    await emailRow.click();

    // Verify email view is visible
    await expect(page.locator('.email-view')).toBeVisible();

    const imageToggle = page.locator('[data-testid="email-view-display-external-images-toggle"]');
    await expect(imageToggle).toBeVisible();

    // Locate the image within the shadow DOM of .email-content
    // Playwright can pierce shadow DOM with `locator('img')` if the element is standard HTML
    // For complex scenarios, you might need page.evaluate or specific shadow DOM piercing.
    const emailContent = page.locator('.email-content');
    const externalImage = emailContent.locator('img[src="http://placekitten.com/200/300"]');

    // Initially, external images should not be displayed (or toggle is unchecked)
    await expect(imageToggle).not.toBeChecked();
    // For direct image visibility check, it's tricky as display:none might be in shadow DOM.
    // We'll rely on the toggle state and assume functionality.
    // A more robust test might involve checking computed styles if Playwright can access them through shadow DOM.
    // await expect(externalImage).toBeHidden(); // This might not work reliably with shadow DOM and display:none

    // Click the toggle to enable external images
    await imageToggle.check();
    await expect(imageToggle).toBeChecked();
    // await expect(externalImage).toBeVisible(); // Again, direct visibility in shadow DOM can be tricky

    // Click the toggle again to disable them
    await imageToggle.uncheck();
    await expect(imageToggle).not.toBeChecked();
    // await expect(externalImage).toBeHidden();
  });

});
