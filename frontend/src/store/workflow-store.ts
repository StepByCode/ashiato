import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ChannelState = "in_progress" | "done";
export type ChannelId = "x" | "instagram" | "facebook";

export type PublicityChannel = {
  id: ChannelId;
  name: string;
  note: string;
  state: ChannelState;
};

type MeetingState = {
  meetingAt: string;
  meetUrl: string;
  setMeetingAt: (value: string) => void;
  setMeetUrl: (url: string) => void;
  resetMeeting: () => void;
};

type PublicityState = {
  template: string;
  channels: PublicityChannel[];
  setTemplate: (value: string) => void;
  updateChannelState: (id: ChannelId, state: ChannelState) => void;
  resetPublicity: () => void;
};

type WorkflowStore = MeetingState & PublicityState;

const MEET_URL_DEFAULT = "https://meet.google.com/abc-defg-hij";
const TEMPLATE_DEFAULT =
  "【イベント告知】4/10(金) 20:00から定例を開催します。参加URLはプロフィールから確認できます。初参加の方も歓迎です。";

const channelDefaults: PublicityChannel[] = [
  { id: "x", name: "X", note: "投稿文の最終チェック", state: "in_progress" },
  { id: "instagram", name: "Instagram", note: "画像差し替えとキャプション確認", state: "in_progress" },
  { id: "facebook", name: "Facebook", note: "イベントページ反映確認", state: "done" },
];

const makeDefaultChannels = () => channelDefaults.map((channel) => ({ ...channel }));

function toLocalDateTimeValue(date: Date) {
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

export function getDefaultMeetingDateTime() {
  const nextMeeting = new Date();
  nextMeeting.setDate(nextMeeting.getDate() + 12);
  nextMeeting.setHours(20, 0, 0, 0);

  return toLocalDateTimeValue(nextMeeting);
}

export const useWorkflowStore = create<WorkflowStore>()(
  persist(
    (set) => ({
      meetingAt: getDefaultMeetingDateTime(),
      meetUrl: MEET_URL_DEFAULT,
      template: TEMPLATE_DEFAULT,
      channels: makeDefaultChannels(),
      setMeetingAt: (meetingAt) => set({ meetingAt }),
      setMeetUrl: (meetUrl) => set({ meetUrl }),
      resetMeeting: () => set({ meetingAt: getDefaultMeetingDateTime(), meetUrl: MEET_URL_DEFAULT }),
      setTemplate: (template) => set({ template }),
      updateChannelState: (id, state) =>
        set((current) => ({
          channels: current.channels.map((channel) => (channel.id === id ? { ...channel, state } : channel)),
        })),
      resetPublicity: () => set({ template: TEMPLATE_DEFAULT, channels: makeDefaultChannels() }),
    }),
    {
      name: "workflow-store-v1",
      version: 1,
    }
  )
);
