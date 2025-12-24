import { Button } from "../ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { oauthLoginUrl } from "../../lib/api";

type OAuthLoginProps = {
  title?: string;
  description?: string;
};

export function OAuthLogin({
  title = "Authentication Required",
  description = "To do this you must be authenticated.",
}: OAuthLoginProps) {
  const onGoogleLogin = () => {
    window.location.href = oauthLoginUrl("google");
  };

  return (
    <Card className="max-w-2xl mx-auto">
      <CardHeader>
        <CardTitle className="text-2xl text-center">{title}</CardTitle>
        <CardDescription className="text-center">{description}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4 items-center">
        <Button
          size="lg"
          variant="outline"
          className="w-full sm:w-auto flex items-center gap-2"
          onClick={onGoogleLogin}
        >
          <GoogleIcon className="h-5 w-5" />
          Log in with Google
        </Button>
      </CardContent>
    </Card>
  );
}

function GoogleIcon({ className = "" }: { className?: string }) {
  // Simple inline Google "G" logo SVG to avoid extra dependencies
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 48 48"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <path fill="#FFC107" d="M43.611 20.083H42V20H24v8h11.303C33.788 31.662 29.273 35 24 35 16.82 35 11 29.18 11 22S16.82 9 24 9c3.411 0 6.509 1.292 8.881 3.419l5.657-5.657C34.871 3.108 29.706 1 24 1 10.745 1 0 11.745 0 25s10.745 24 24 24 24-10.745 24-24c0-1.627-.168-3.215-.389-4.917z"/>
      <path fill="#FF3D00" d="M6.306 14.691l6.571 4.816C14.484 16.262 18.87 13 24 13c3.411 0 6.509 1.292 8.881 3.419l5.657-5.657C34.871 3.108 29.706 1 24 1 15.317 1 7.83 5.821 3.694 12.691z"/>
      <path fill="#4CAF50" d="M24 49c5.169 0 9.86-1.977 13.409-5.197l-6.192-5.238C29.024 40.839 26.619 41.8 24 41.8 18.758 41.8 14.262 38.412 12.303 33.662l-6.54 5.034C9.858 44.991 16.457 49 24 49z"/>
      <path fill="#1976D2" d="M43.611 20.083H42V20H24v8h11.303c-1.196 3.48-3.862 6.193-7.086 7.467.002-.001.004-.001.006-.002l6.192 5.238C31.927 42.043 36.5 45 42 45c1.657 0 3.25-.254 4.75-.729C47.832 41.586 49 38.41 49 35c0-1.627-.168-3.215-.389-4.917z"/>
    </svg>
  );
}

export default OAuthLogin;
