/// <reference types="vite/client" />

declare module "@novnc/novnc" {
  export default class RFB {
    constructor(
      target: HTMLElement,
      url: string,
      options?: Record<string, unknown>,
    );
    scaleViewport: boolean;
    resizeSession: boolean;
    viewOnly: boolean;
    disconnect(): void;
    addEventListener(name: string, listener: (event: Event) => void): void;
  }
}
