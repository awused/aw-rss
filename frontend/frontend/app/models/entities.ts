export interface Category {
  readonly commitTimestamp: number;
  readonly disabled: boolean;
  readonly hidden: boolean;
  readonly id: number;
}

export interface Feed {
  readonly commitTimestamp: number;
  readonly createTimestamp: number;
  readonly disabled: boolean;
  readonly failingSince?: string;
  readonly id: number;
  readonly siteUrl: string;
  readonly title: string;
  // "url" might not be a URL. If it starts with ! it's a shell command.
  readonly url: string;
  // If the user has overridden the title with their own setting
  readonly userTitle?: string;
}

export interface Item {
  readonly commitTimestamp: number;
  readonly description?: string;
  readonly feedId: number;
  readonly id: number;
  readonly read: boolean;
  readonly timestamp: string;
  readonly title: string;
  readonly url: string;
}
