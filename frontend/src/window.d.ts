declare global {
  interface Window {
    OPENGYM_CONFIG?: {
      API_BASE_URL?: string;
      IS_DEMO_MODE: boolean;
    };
  }
}

export {};
