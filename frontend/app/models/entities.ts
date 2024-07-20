export interface Category {
  readonly commitTimestamp: number;
  readonly disabled: boolean;
  // Items in this feed are hidden in the main view
  readonly hiddenMain: boolean;
  // This category and its feeds are hidden in the nav bar
  readonly hiddenNav: boolean;
  readonly id: number;
  // A short name consisting of alphabetic characters and hyphens
  // Used in routes
  readonly name: string;
  readonly sortPosition?: number;
  readonly title: string;
}

export const CATEGORY_NAME_REGEX = /^[a-z][a-z0-9-]+$/;

export interface Feed {
  readonly categoryId?: number;
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
  readonly feedId: number;
  readonly id: number;
  readonly read: boolean;
  readonly timestamp: string;
  readonly title: string;
  readonly url: string;
}
