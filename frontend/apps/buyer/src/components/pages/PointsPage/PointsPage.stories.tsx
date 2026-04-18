import type { Meta, StoryObj } from "@storybook/react";
import { PointsBalanceCardPresenter } from "./PointsBalanceCard.presenter";
import { PointsTransactionsListPresenter } from "./PointsTransactionsList.presenter";
import { CouponRedemptionsListPresenter } from "./CouponRedemptionsList.presenter";

const balanceLabels = {
  spendable: "利用可能ポイント",
  pending: "保留中",
  lifetimeEarned: "累計獲得",
  lifetimeRedeemed: "累計利用",
  pointsUnit: "pt",
};

const txLabels = {
  date: "日付",
  type: "種別",
  amount: "増減",
  balanceAfter: "残高",
  note: "メモ",
};

const txTypeLabels = {
  earn: "獲得",
  redeem: "利用",
  refund: "返金",
  reverse_earn: "獲得取消",
  adjust: "調整",
  expire: "失効",
} as const;

const meta: Meta<typeof PointsBalanceCardPresenter> = {
  component: PointsBalanceCardPresenter,
  title: "Pages/PointsPage/BalanceCard",
};
export default meta;

type BalanceStory = StoryObj<typeof PointsBalanceCardPresenter>;

export const ZeroBalance: BalanceStory = {
  args: {
    spendable: 0,
    pendingRedemption: 0,
    lifetimeEarned: 0,
    lifetimeRedeemed: 0,
    labels: balanceLabels,
  },
};

export const WithPending: BalanceStory = {
  args: {
    spendable: 1500,
    pendingRedemption: 200,
    lifetimeEarned: 5000,
    lifetimeRedeemed: 3300,
    labels: balanceLabels,
  },
};

export const TransactionsEmpty: StoryObj<typeof PointsTransactionsListPresenter> = {
  render: (args) => <PointsTransactionsListPresenter {...args} />,
  args: {
    transactions: [],
    loaded: true,
    emptyLabel: "履歴はまだありません",
    loadingLabel: "...",
    typeLabels: txTypeLabels,
    columns: txLabels,
    locale: "ja",
  },
};

export const TransactionsMixed: StoryObj<typeof PointsTransactionsListPresenter> = {
  render: (args) => <PointsTransactionsListPresenter {...args} />,
  args: {
    loaded: true,
    emptyLabel: "",
    loadingLabel: "",
    typeLabels: txTypeLabels,
    columns: txLabels,
    locale: "ja",
    transactions: [
      {
        id: "t1",
        buyer_auth0_id: "auth0|1",
        type: "earn",
        amount: 80,
        balance_after: 80,
        source_type: "order_paid",
        source_id: "o1",
        note: "ご注文 #o1",
        created_at: "2026-04-01T00:00:00Z",
      },
      {
        id: "t2",
        buyer_auth0_id: "auth0|1",
        type: "redeem",
        amount: -50,
        balance_after: 30,
        source_type: "reservation_commit",
        source_id: "r1",
        note: "",
        created_at: "2026-04-02T00:00:00Z",
      },
    ],
  },
};

export const CouponHistoryEmpty: StoryObj<typeof CouponRedemptionsListPresenter> = {
  render: (args) => <CouponRedemptionsListPresenter {...args} />,
  args: {
    redemptions: [],
    loaded: true,
    emptyLabel: "クーポン利用履歴はまだありません",
    loadingLabel: "...",
    columns: {
      date: "日付",
      orderId: "注文番号",
      discount: "割引",
      refunded: "返金",
      status: "状態",
    },
    statusLabels: {
      active: "適用中",
      refundedFully: "全額返金",
      refundedPartially: "一部返金",
    },
    locale: "ja",
  },
};
