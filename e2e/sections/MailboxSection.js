'use strict';

/**
 * Left-pane mailbox navigation.
 * The list is collapsed by default and populated lazily on first expand.
 */
class MailboxSection {
  constructor(page) {
    this.page          = page;
    this.allEmailsLink = page.locator('[data-testid="all-emails-link"]');
    this.toggleButton  = page.locator('[data-testid="mailbox-toggle"]');
    this.list          = page.locator('#mailboxList');
  }

  /** Locator for a specific mailbox item by recipient address. */
  item(address) {
    return this.page.locator(`[data-testid="mailbox-item-${address}"]`);
  }

  /**
   * Expand the mailbox list; waits for the /api/mailboxes response
   * and for the list to become visible.
   */
  async expand() {
    const resp = this.page.waitForResponse(r => r.url().includes('/api/mailboxes'));
    await this.toggleButton.click();
    await resp;
    await this.list.waitFor({ state: 'visible' });
  }

  /**
   * Click a mailbox item by recipient address; waits for the filtered
   * email list to reload.
   */
  async clickMailbox(address) {
    const resp = this.page.waitForResponse(r => r.url().includes('/api/emails'));
    await this.item(address).click();
    await resp;
  }

  /** Click "All" to reset the filter and show every email. */
  async showAll() {
    const resp = this.page.waitForResponse(r => r.url().includes('/api/emails'));
    await this.allEmailsLink.click();
    await resp;
  }
}

module.exports = { MailboxSection };
