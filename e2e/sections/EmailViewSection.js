'use strict';

const { BodyVersionsSection } = require('./BodyVersionsSection');

class EmailViewSection {
  constructor(page) {
    this.page = page;
    /** Locators exposed for direct assertions in tests */
    this.locator               = page.locator('.email-view');
    this.backButton            = page.locator('[data-testid="email-view-back-button"]');
    this.releaseButton         = page.locator('[data-testid="email-view-release-button"]');
    this.deleteButton          = page.locator('[data-testid="email-view-delete-button"]');
    this.header                = page.locator('.email-header');
    this.content               = page.locator('.email-content');
    this.externalImagesToggle  = page.locator('[data-testid="email-view-display-external-images-toggle"]');
    this.attachmentsList       = page.locator('[data-testid="email-attachments-list"]');
    this.attachmentLinks       = page.locator('[data-testid^="attachment-link-"]');
    /** Sub-section for body version tab switching */
    this.bodyVersions          = new BodyVersionsSection(page);
  }

  // ── Actions ──────────────────────────────────────────────────────────────

  /** Navigate back to the email list. */
  async goBack() {
    await this.backButton.click();
  }

  /** Open the release modal from the email view header. */
  async openReleaseModal() {
    await this.releaseButton.click();
  }

  /**
   * Delete the currently open email via the email-view delete button.
   * Waits for the DELETE response.
   */
  async delete() {
    const resp = this.page.waitForResponse(
      r => r.url().includes('/api/emails/') && r.request().method() === 'DELETE'
    );
    await this.deleteButton.click();
    await resp;
  }
}

module.exports = { EmailViewSection };
