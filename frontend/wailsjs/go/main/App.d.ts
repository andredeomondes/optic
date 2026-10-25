// Types for the Go App struct (app.go). Keep in sync when changing exported methods.

export type TV = { name: string; model: string; ip: string; remotev2: boolean };
export type DeviceInfo = { model: string; vendor: string; version: string; appVersion: string };

export function CancelPairing(): Promise<void>;
export function Connect(arg1: string): Promise<DeviceInfo>;
export function Disconnect(): Promise<void>;
export function Discover(): Promise<Array<TV>>;
export function FinishPairing(arg1: string): Promise<void>;
export function LastHost(): Promise<string>;
export function LaunchApp(arg1: string): Promise<void>;
export function PowerOn(arg1: string): Promise<void>;
export function SendKey(arg1: string): Promise<void>;
export function SendText(arg1: string): Promise<void>;
export function SetVolume(arg1: number): Promise<void>;
export function StartPairing(arg1: string): Promise<string>;
export function CloseToTray(): Promise<boolean>;
export function QuitApp(): Promise<void>;
export function ResizeWindow(arg1: number, arg2: number, arg3: number, arg4: number): Promise<void>;
export function SetCloseToTray(arg1: boolean): Promise<void>;
export function SetCompactMode(arg1: boolean): Promise<void>;
export function ShowWindow(): Promise<void>;
export function ToggleFullscreen(): Promise<boolean>;
export function TrayAvailable(): Promise<boolean>;
