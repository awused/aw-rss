import {Feed,
        Item} from './entities';

export const FeedFixtures: {[x: string]: Feed} = {
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


export const ItemFixtures: {[x: string]: Item} = {
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
