import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { PointsTransactionType } from "@ec-marketplace/types";
import { PointsTransactionsListPresenter } from "./PointsTransactionsList.presenter";

const typeLabels: Record<PointsTransactionType, string> = {
  earn: "獲得",
  redeem: "利用",
  refund: "返金",
  reverse_earn: "獲得取消",
  adjust: "調整",
  expire: "失効",
};

const columns = {
  date: "日付",
  type: "種別",
  amount: "増減",
  balanceAfter: "残高",
  note: "メモ",
};

describe("PointsTransactionsListPresenter", () => {
  it("renders the loading state when not loaded", () => {
    render(
      <PointsTransactionsListPresenter
        transactions={[]}
        emptyLabel="ありません"
        loadingLabel="読み込み中"
        loaded={false}
        typeLabels={typeLabels}
        columns={columns}
        locale="ja"
      />
    );
    expect(screen.getByText("読み込み中")).toBeInTheDocument();
  });

  it("renders the empty state when loaded with no transactions", () => {
    render(
      <PointsTransactionsListPresenter
        transactions={[]}
        emptyLabel="履歴はまだありません"
        loadingLabel="..."
        loaded
        typeLabels={typeLabels}
        columns={columns}
        locale="ja"
      />
    );
    expect(screen.getByText("履歴はまだありません")).toBeInTheDocument();
  });

  it("renders earn rows in green and redeem rows in red", () => {
    render(
      <PointsTransactionsListPresenter
        transactions={[
          {
            id: "t1",
            buyer_auth0_id: "auth0|1",
            type: "earn",
            amount: 100,
            balance_after: 100,
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
            balance_after: 50,
            source_type: "reservation_commit",
            source_id: "r1",
            note: "",
            created_at: "2026-04-02T00:00:00Z",
          },
        ]}
        emptyLabel=""
        loadingLabel=""
        loaded
        typeLabels={typeLabels}
        columns={columns}
        locale="ja"
      />
    );
    expect(screen.getByText("獲得")).toBeInTheDocument();
    expect(screen.getByText("利用")).toBeInTheDocument();
    expect(screen.getByText("+100")).toBeInTheDocument();
    expect(screen.getByText("-50")).toBeInTheDocument();
  });
});
