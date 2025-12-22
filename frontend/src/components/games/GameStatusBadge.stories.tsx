import type { Story } from "@ladle/react";
import { GameStatusBadge } from "./GameStatusBadge";

export const Published: Story = () => <GameStatusBadge state="published" />;

export const Scheduled: Story = () => <GameStatusBadge state="scheduled" />;

export const Draft: Story = () => <GameStatusBadge state="draft" />;
