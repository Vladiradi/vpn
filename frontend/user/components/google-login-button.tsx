"use client";

import { useGoogleLogin } from "@react-oauth/google";
import { Button } from "@/components/ui/button";

interface GoogleLoginButtonProps {
  label: string;
  disabled?: boolean;
  onSuccess: (accessToken: string) => void;
  onError: () => void;
}

// Rendered only when NEXT_PUBLIC_GOOGLE_CLIENT_ID is set, so the hook always
// runs inside GoogleOAuthProvider.
export function GoogleLoginButton({ label, disabled, onSuccess, onError }: GoogleLoginButtonProps) {
  const google = useGoogleLogin({
    onSuccess: (tokenResponse) => onSuccess(tokenResponse.access_token),
    onError,
  });

  return (
    <Button variant="outline" className="w-full" onClick={() => google()} disabled={disabled}>
      {label}
    </Button>
  );
}
