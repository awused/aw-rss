import { MainViewModule } from './main-view.module';

describe('MainViewModule', () => {
  let itemListModule: MainViewModule;

  beforeEach(() => {
    itemListModule = new MainViewModule();
  });

  it('should create an instance', () => {
    expect(itemListModule).toBeTruthy();
  });
});
