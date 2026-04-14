export {};

declare global {
  interface Window {
    siby: SibyAPI;
  }
}

interface SibyAPI {
  version: string;
  creator: string;
  
  initialize(): Promise<void>;
  
  ask(query: string): Promise<string>;
  
  scan(): Promise<ScanResult>;
  
  activateGodMode(): Promise<GodModeStatus>;
  
  runEvolution(): Promise<EvolutionReport>;
  
  syncCloud(): Promise<SyncStatus>;
  
  on(event: string, callback: Function): void;
  
  off(event: string, callback: Function): void;
}

interface ScanResult {
  files: number;
  lines: number;
  dependencies: string[];
  summary: string;
}

interface GodModeStatus {
  active: boolean;
  cpu: number;
  memory: number;
  uptime: string;
}

interface EvolutionReport {
  lessonsLearned: number;
  optimizationsApplied: number;
  topicsProgress: Record<string, number>;
}

interface SyncStatus {
  state: string;
  lastSync: string;
  pendingChanges: number;
}
