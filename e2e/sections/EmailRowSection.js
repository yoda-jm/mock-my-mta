'use strict';

/**
 * Represents a single row in the email list table.
 * Construct via EmailListSection.row(index) or .firstRow().
 */
class EmailRowSection {
  constructor(page, rowLocator) {
    this.page = page;
    this.locator = rowLocator;
    /** Locators exposed for direct assertions in tests */
    this.fromCell      = rowLocator.locator('[data-testid^="email-from-"]');
    this.previewCell   = rowLocator.locator('[data-testid^="email-preview-"]');
    this.deleteButton  = rowLocator.locator('[data-testid^="email-delete-button-"]');
    this.releaseButton = rowLocator.locator('[data-testid^="email-release-button-"]');
    this.attachmentIcon = rowLocator.locator('[data-testid^="email-attachment-icon-"] i.bi-paperclip');
  }

  /** Click the row to open the email view; waits for the body API call. */
  async open() {
    const resp = this.page.waitForResponse(r => r.url().includes('/body/'));
    await this.locator.click();
    await resp;
  }

  /**
   * Delete this email from the row action button.
   * Waits for both the DELETE and the subsequent GET (list refresh).
   */
  async delete() {
    const deleteResp  = this.page.waitForResponse(
      r => r.url().includes('/api/emails/') && r.request().method() === 'DELETE'
    );
    const refreshResp = this.page.waitForResponse(
      r => r.url().includes('/api/emails') && r.request().method() === 'GET'
    );
    await this.deleteButton.click();
    await deleteResp;
    await refreshResp;
  }

  /** Click the release button (opens the release modal). */
  async openReleaseModal() {
    await this.releaseButton.click();
  }

  async hasAttachment() {
    return this.attachmentIcon.isVisible();
  }
}

module.exports = { EmailRowSection };
