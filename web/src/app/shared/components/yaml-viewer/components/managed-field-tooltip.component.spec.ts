import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ManagedFieldTooltipComponent } from './managed-field-tooltip.component';

describe('ManagedFieldTooltipComponent', () => {
  let component: ManagedFieldTooltipComponent;
  let fixture: ComponentFixture<ManagedFieldTooltipComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ManagedFieldTooltipComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(ManagedFieldTooltipComponent);
    component = fixture.componentInstance;

    fixture.componentRef.setInput('manager', 'test-manager');
    fixture.componentRef.setInput('time', 1609459200000000000n);
    fixture.componentRef.setInput('timezoneShift', 9);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
