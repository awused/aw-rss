export interface Feed {
  id: number;
  url: string;
  title: string;
  userTitle: string;
  siteUrl: string;
  disabled: boolean;
  lastFetchFailed: boolean;
}

export interface Item {
  id: number;
  feedId: number;
  title: string;
  url: string;
  timestamp: string;
  read: boolean;
}
