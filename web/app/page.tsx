"use client";

import { useEffect, useState } from "react";
import { getCurrentUser, type Me } from "@/lib/api";
import { SignedOutLanding } from "@/components/SignedOutLanding";
import { Shell } from "@/components/Shell";

export default function Home() {
  const [me, setMe] = useState<Me | null | undefined>(undefined);

  useEffect(() => {
    getCurrentUser().then(setMe);
  }, []);

  if (me === undefined) {
    return null;
  }
  if (me === null) {
    return <SignedOutLanding />;
  }
  return <Shell me={me} />;
}
