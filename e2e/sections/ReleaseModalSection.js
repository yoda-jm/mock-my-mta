'use strict';

class ReleaseModalSection {
  constructor(page) {
    this.locator = page.locator('#releaseEmailModal');
    /** Locators exposed for direct assertions in tests */
    this.emailIdInput         = this.locator.locator('[data-testid="release-modal-email-id-input"]');
    this.relayConfigSelect    = this.locator.locator('[data-testid="release-modal-relay-config-select"]');
    this.senderOriginalRadio  = this.locator.locator('[data-testid="release-modal-sender-original-radio"]');
    this.overrideSenderInput  = this.locator.locator('[data-testid="release-modal-override-sender-input"]');
    this.receiversOriginalRadio = this.locator.locator('[data-testid="release-modal-receivers-original-radio"]');
    this.overrideReceiversInput = this.locator.locator('[data-testid="release-modal-override-receivers-input"]');
    this.releaseButton        = this.locator.locator('[data-testid="release-modal-release-button"]');
    this.closeButton          = this.locator.locator('[data-testid="release-modal-close-button"]');
    this.closeButtonX         = this.locator.locator('[data-testid="release-modal-close-button-x"]');
  }

  /** Close via the footer "Close" button. */
  async close() {
    await this.closeButton.click();
  }

  /** Close via the header "×" button. */
  async closeX() {
    await this.closeButtonX.click();
  }
}

module.exports = { ReleaseModalSection };
