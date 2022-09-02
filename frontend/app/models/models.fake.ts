import {Feed,
        Item} from './entities';

export const FEED_FIXTURES: {[x: string]: Feed} = {
  emptyFeed: {
    commitTimestamp: 0,
    createTimestamp: 0,
    disabled: false,
    id: 0,
    siteUrl: '',
    title: '',
    url: '',
    userTitle: '',
  }
};


export const ITEM_FIXTURES: {[x: string]: Item} = {
  emptyItem: {
    commitTimestamp: 0,
    feedId: 0,
    id: 0,
    read: false,
    timestamp: '',
    title: '',
    url: '',
  }
};
