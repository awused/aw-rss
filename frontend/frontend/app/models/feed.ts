export interface Feed {
  commitTimestamp: number;
  createTimestamp: number;
  disabled: boolean;
  id: number;
  lastFetchFailed: boolean;
  lastSuccessTime: string;
  siteUrl: string;
  title: string;
  // "url" might not be a URL. If it starts with ! it's a shell command.
  url: string;
  userTitle: string;
}
