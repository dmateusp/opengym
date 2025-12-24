import type { StoryDefault } from '@ladle/react';
import OAuthLogin from './OAuthLogin';

export default {
  title: 'Auth/OAuthLogin',
} satisfies StoryDefault;

export const Default = () => <OAuthLogin />;

export const CustomText = () => (
  <OAuthLogin
    title="Join a game"
    description="To do this you must be authenticated. Log in below."
  />
);
