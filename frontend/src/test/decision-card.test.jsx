// decision-card.test.js — DecisionCard + DocumentReviewCard 測試
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import DecisionCard from '../components/DecisionCard';
import DocumentReviewCard from '../components/DocumentReviewCard';

describe('DecisionCard', () => {
  it('renders title and three action buttons', () => {
    render(
      <DecisionCard
        cardId="test-1"
        title="測試卡片"
        description="這是測試"
        onDecision={() => {}}
      />
    );
    expect(screen.getByTestId('decision-card')).toBeInTheDocument();
    expect(screen.getByText('測試卡片')).toBeInTheDocument();
    expect(screen.getByTestId('decision-agree')).toBeInTheDocument();
    expect(screen.getByTestId('decision-disagree')).toBeInTheDocument();
    expect(screen.getByTestId('decision-custom-toggle')).toBeInTheDocument();
  });

  it('calls onDecision with agree', () => {
    const handler = vi.fn();
    render(
      <DecisionCard cardId="c1" title="T" onDecision={handler} />
    );
    fireEvent.click(screen.getByTestId('decision-agree'));
    expect(handler).toHaveBeenCalledWith('c1', 'agree');
    expect(screen.getByTestId('decision-card-decided')).toBeInTheDocument();
  });

  it('calls onDecision with disagree', () => {
    const handler = vi.fn();
    render(
      <DecisionCard cardId="c2" title="T" onDecision={handler} />
    );
    fireEvent.click(screen.getByTestId('decision-disagree'));
    expect(handler).toHaveBeenCalledWith('c2', 'disagree');
  });

  it('shows custom input on toggle and submits', () => {
    const handler = vi.fn();
    render(
      <DecisionCard cardId="c3" title="T" onDecision={handler} />
    );
    fireEvent.click(screen.getByTestId('decision-custom-toggle'));
    expect(screen.getByTestId('decision-custom-input')).toBeInTheDocument();

    const textarea = screen.getByPlaceholderText('請輸入您的意見...');
    fireEvent.change(textarea, { target: { value: '我的意見' } });
    fireEvent.click(screen.getByText('送出'));
    expect(handler).toHaveBeenCalledWith('c3', 'custom', '我的意見');
  });

  it('shows preview toggle when preview prop provided', () => {
    render(
      <DecisionCard cardId="c4" title="T" preview="前 500 字..." onDecision={() => {}} />
    );
    expect(screen.getByText('展開預覽 ▼')).toBeInTheDocument();
    fireEvent.click(screen.getByText('展開預覽 ▼'));
    expect(screen.getByText('前 500 字...')).toBeInTheDocument();
  });

  it('displays custom labels and hints', () => {
    render(
      <DecisionCard
        cardId="c5"
        title="T"
        agreeLabel="確認儲存"
        disagreeLabel="取消"
        agreeReason="存入專案"
        disagreeConsequence="不保存"
        onDecision={() => {}}
      />
    );
    expect(screen.getByText('確認儲存')).toBeInTheDocument();
    expect(screen.getByText('取消')).toBeInTheDocument();
    expect(screen.getByText('（存入專案）')).toBeInTheDocument();
    expect(screen.getByText('（不保存）')).toBeInTheDocument();
  });
});

describe('DocumentReviewCard', () => {
  it('renders nothing when no pending documents', () => {
    const { container } = render(<DocumentReviewCard onToast={() => {}} />);
    expect(container.querySelector('.document-review-cards')).toBeNull();
  });
});
