import { ItemListModule } from './item-list.module';

describe('ItemListModule', () => {
  let itemListModule: ItemListModule;

  beforeEach(() => {
    itemListModule = new ItemListModule();
  });

  it('should create an instance', () => {
    expect(itemListModule).toBeTruthy();
  });
});
