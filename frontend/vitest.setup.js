import { vi } from 'vitest';

// Mock for HTMLElement.attachInternals used by Material Web Components
if (typeof HTMLElement !== 'undefined' && !HTMLElement.prototype.attachInternals) {
  HTMLElement.prototype.attachInternals = vi.fn(() => ({
    setFormValue: vi.fn(),
  }));
}

// Mock for the Wails backend bridge
global.window.go = {
  app: {
    App: {
      ListMIBModules: vi.fn(),
      GetMIBStats: vi.fn(),
      LoadMIBFile: vi.fn(),
      DeleteMIBModule: vi.fn(),
      GetMIBModuleDetails: vi.fn(),
      GetMIBTree: vi.fn(),
      SNMPGet: vi.fn(),
      SNMPGetNext: vi.fn(),
      SNMPWalk: vi.fn(),
      SNMPGetBulk: vi.fn(),
      SNMPSet: vi.fn(),
      GetMIBNode: vi.fn(),
      ListHosts: vi.fn(),
      DeleteHost: vi.fn(),
      SaveCSVFile: vi.fn(),
      AddBookmark: vi.fn(),
      CreateBookmarkFolder: vi.fn(),
      DeleteBookmarkFolder: vi.fn(),
      MoveBookmark: vi.fn(),
      MoveBookmarkFolder: vi.fn(),
      RemoveBookmark: vi.fn(),
      RenameBookmarkFolder: vi.fn(),
    },
  },
};
